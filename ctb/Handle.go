package ctb

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"image"
	"os"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Handle CTB查询句柄
type Handle struct {
	db          *gorm.DB
	CtbFilepath string
}

func NewHandle(filepath, returnedImagePathPrefix string) *Handle {
	// 不打印sql
	newLogger := logger.New(
		log.StandardLogger(), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,          // Disable color
		},
	)
	// create sqlite handle
	db, err := gorm.Open(sqlite.Open(filepath), &gorm.Config{
		PrepareStmt: true,
		Logger:      newLogger,
	})
	if err != nil {
		log.Errorf("An error occurred while creating the sqlite database handle for %v: %v", filepath, err)
		return nil
	}
	handle := &Handle{
		db:          db,
		CtbFilepath: filepath,
	}
	return handle
}

// GetTotalNodesCount 获取节点数量
func (r Handle) GetTotalNodesCount() (int64, error) {
	var a []tNode
	result := r.db.Find(&a)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		} else {
			return 0, result.Error
		}
	}
	return int64(len(a)), nil
}

func (r Handle) GetNodeById(id int32) (*CtNode, error) {
	nm, err := r.selectNodeMetaById(id)
	if err != nil {
		return nil, err
	}
	c, err := r.selectChildrenByFatherId(id)
	return NewCtNode(&nm, len(c) > 0), nil
}

func (r Handle) findRootNodeById(id int32) (*CtNode, error) {
	c, err := r.selectChildrenByNodeId(id)
	if err != nil {
		return nil, err
	}
	for c.FatherId != 0 {
		c, err = r.selectChildrenByNodeId(c.NodeId)
		if err != nil {
			return nil, err
		}
	}
	n, err := r.selectNodeMetaById(c.NodeId)
	if err != nil {
		return nil, err
	}
	nc, err := r.selectChildrenByFatherId(n.NodeId)
	if err != nil {
		return nil, err
	}
	return NewCtNode(&n, len(nc) > 0), nil
}

func (r Handle) GetSubNodesById(id int32) ([]*CtNode, error) {
	list, err := r.selectChildrenByFatherId(id)
	if err != nil {
		return nil, err
	}
	var ret []*CtNode
	for _, c := range list {
		nm, err := r.GetNodeById(c.NodeId)
		if err != nil {
			return nil, err
		}
		ret = append(ret, nm)
	}
	return ret, nil
}

// GetNodeContentById 如果指定了路径则保存图片和附件到磁盘，并返回访问路径，否则以[]byte的形式保存图片和附件
func (r Handle) GetNodeContentById(id int32, pathToSaveBinary *string) (*CtNodeContent, error) {
	meta, err := r.GetNodeById(id)
	if err != nil {
		return nil, err
	}
	raw, err := r.selectNodeContentById(id)
	if err != nil {
		return nil, err
	}
	ret := CtNodeContent{
		Id:         raw.NodeId,
		CreateTime: raw.TsCreation,
		UpdateTime: raw.TsLastsave,
		IsRichText: meta.IsRichText,
	}
	// code box 不需要处理富文本
	if !ret.IsRichText {
		ret.Language = raw.Syntax
		ret.Code = raw.Txt
		return &ret, nil
	}
	// 准备 anchored widgets
	var anchoredWidgets []CtAnchoredWidget
	{
		// code-box
		var codeBoxes []tCodeBox
		r.db.Where("node_id = ?", id).Find(&codeBoxes)
		for _, codeBox := range codeBoxes {
			anchoredWidgets = append(anchoredWidgets, NewCtCodeBox(&codeBox))
		}
		// grid
		var tables []tGrid
		r.db.Where("node_id = ?", id).Find(&tables)
		for _, grid := range tables {
			anchoredWidgets = append(anchoredWidgets, NewCtTable(&grid))
		}
		// images: png / embfile / anchor / latex
		images, err := r.selectImagesByNodeId(id)
		if err != nil {
			return nil, err
		}
		for _, img := range images {
			// anchor
			if img.Anchor != "" {
				anchoredWidgets = append(anchoredWidgets, NewCtAnchor(img))
				continue
			}
			// latex
			if img.Filename == "__ct_special.tex" {
				// TODO Latex
				continue
			}
			// 不需要保存图片和嵌入式附件到磁盘
			if pathToSaveBinary == nil {
				if img.Filename != "" {
					// png
					_png, _, err := image.DecodeConfig(bytes.NewReader(img.Png))
					if err != nil {
						return nil, err
					}
					anchoredWidgets = append(anchoredWidgets, NewCtPng(img, _png.Width, _png.Height, nil))
				} else {
					// embfile
					anchoredWidgets = append(anchoredWidgets, NewCtEmbFile(img, nil))
				}
				continue
			}
			// 需要保存图片与附件到磁盘：确保保存附件的文件夹存在
			_ = os.Mkdir(*pathToSaveBinary, 0755)
			_, err := os.Stat(*pathToSaveBinary)
			if err != nil {
				return nil, err
			}
			// embfile or png
			var (
				filename string
				filepath string
			)
			if img.Filename != "" {
				// embfile
				filename = fmt.Sprintf("%d_%d%s", img.NodeId, img.Offset, path.Ext(img.Filename))
				filepath = path.Join(*pathToSaveBinary, filename)
				anchoredWidgets = append(anchoredWidgets, NewCtEmbFile(img, &filepath))
			} else {
				// png
				filename = fmt.Sprintf("%d_%d.png", img.NodeId, img.Offset)
				filepath = path.Join(*pathToSaveBinary, filename)
				_png, _, err := image.DecodeConfig(bytes.NewReader(img.Png))
				if err != nil {
					return nil, err
				}
				anchoredWidgets = append(anchoredWidgets, NewCtPng(img, _png.Width, _png.Height, &filepath))
			}
			// 写文件
			file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				_ = file.Close()
				return nil, err
			}
			_, err = file.Write(img.Png)
			if err != nil {
				_ = file.Close()
				return nil, err
			}
		}
		// 按 offset 排序所有 anchored widgets
		sort.Slice(anchoredWidgets, func(i, j int) bool {
			return anchoredWidgets[i].GetOffset() < anchoredWidgets[j].GetOffset()
		})
	}
	/*
		解析 xml 内容
		tip: '\n' is the only separator of each line
	*/
	var xmlDocument XmlDocument
	texts := [][]*CtText{{}} // 富文本段落集合；每行代表一个段落（也就是一行）。初始化一个空行。
	currentLineIndex := 0
	{
		err := xml.Unmarshal([]byte(raw.Txt), &xmlDocument)
		if err != nil {
			panic(err)
		}
		for _, e := range xmlDocument.RichTexts {
			richTextNode := NewCtText(&e)
			// 没有内容的功能性节点
			if richTextNode.Text == "" {
				texts[currentLineIndex] = append(texts[currentLineIndex], richTextNode)
				continue
			}
			// 剩余内容分行处理
			ls := strings.Split(richTextNode.Text, "\n")
			for len(ls) > 0 {
				curLineText := *richTextNode // 拷贝（包括文本样式等）
				curLineText.Text = ls[0]     // 只保留当前行文本
				// 将非空内容添加到当前行（抛弃空文本，因为文本样式等后续部分都有）
				if curLineText.Text != "" {
					texts[currentLineIndex] = append(texts[currentLineIndex], &curLineText)
				}
				ls = ls[1:]
				// 如果还有剩余行，则需要为下一行初始化空数组
				if len(ls) > 0 {
					texts = append(texts, []*CtText{})
					currentLineIndex++
				}
			}
		}
	}
	// 与 anchored widgets 组合到一起
	resultSet := [][]interface{}{{}} // 结果集，初始化一个空行
	currentLineIndex = 0
	var chars int32 = 0 // 记录当前字符数（配合偏移量来插入 anchored widget）
	for len(texts) > 0 {
		// 如果 anchored widgets 没了的话，把剩余的 text 拼接到后面
		if len(anchoredWidgets) == 0 {
			for _, line := range texts {
				for _, el := range line {
					resultSet[currentLineIndex] = append(resultSet[currentLineIndex], el)
				}
				currentLineIndex++
				texts = texts[1:]
				if len(texts) > 0 {
					resultSet = append(resultSet, []interface{}{}) // 为下一行初始化空数组
				}
			}
			texts = nil
			break
		}
		// 按偏移量处理 anchored widget
		offset := anchoredWidgets[0].GetOffset()
		// 字符数量刚好符合偏移量
		if offset == chars && (len(texts[0]) == 0 || texts[0][0].Text != "") { // 跳过功能性空串
			resultSet[currentLineIndex] = append(resultSet[currentLineIndex], anchoredWidgets[0]) // 添加到当前行
			anchoredWidgets = anchoredWidgets[1:]
			chars++
			continue
		}
		// 行末
		if len(texts[0]) == 0 {
			texts = texts[1:]
			currentLineIndex++
			resultSet = append(resultSet, []interface{}{})
			chars++ // 隐含的 '\n'
			continue
		}
		// 处理一块富文本
		curTextPtr := texts[0][0]
		curTextLen := int32(utf8.RuneCountInString(curTextPtr.Text))
		{
			// 功能性空串
			if curTextLen == 0 {
				resultSet[currentLineIndex] = append(resultSet[currentLineIndex], curTextPtr)
				texts[0] = texts[0][1:]
				continue
			}
			// 字符数量已经超过偏移量
			if offset < chars {
				errInfo := fmt.Sprintf("expected offset will not appear: cur=%d expected=%d\n", chars, offset)
				errInfo += "remained texts in current line:\n"
				for _, t := range texts[currentLineIndex] {
					errInfo += t.Text
				}
				errInfo += "\n"
				if len(texts)-1 == currentLineIndex {
					errInfo += "no more lines."
				} else {
					errInfo += "texts in next line:\n"
					for _, t := range texts[currentLineIndex+1] {
						errInfo += t.Text
					}
				}
				return nil, errors.New(errInfo)
			}
			if offset > chars {
				// 期待的偏移量在后面
				//         ↓anchored-widget
				// text
				if chars+curTextLen < offset {
					resultSet[currentLineIndex] = append(resultSet[currentLineIndex], curTextPtr)
					chars += curTextLen
					texts[0] = texts[0][1:]
					continue
				}
				// 期待的偏移量在这段文本的内部（当前文本要拆成两块）
				//   ↓anchored-widget
				// text
				if chars+curTextLen > offset {
					leftText := *curTextPtr                                         // 拷贝
					leftText.Text = string([]rune(curTextPtr.Text)[:offset-chars])  // 保留左半部分
					rightText := *curTextPtr                                        // 拷贝
					rightText.Text = string([]rune(curTextPtr.Text)[offset-chars:]) // 保留右半部分
					texts[0] = texts[0][1:]                                         //
					// 左半部分
					resultSet[currentLineIndex] = append(resultSet[currentLineIndex], &leftText)
					chars = offset
					// 拼接 anchored widget
					resultSet[currentLineIndex] = append(resultSet[currentLineIndex], &anchoredWidgets[0])
					anchoredWidgets = anchoredWidgets[1:]
					chars++
					// 右半部分推回去，假装什么都没发生
					texts[0] = append([]*CtText{&rightText}, texts[0]...)
					continue
				}
				// 刚好在当前文本块的后面
				//     ↓special-element
				// text
				if chars+curTextLen == offset {
					// 文本块 拼接到结果集的当前行
					resultSet[currentLineIndex] = append(resultSet[currentLineIndex], curTextPtr)
					chars += curTextLen
					texts[0] = texts[0][1:]
				}
			}
		}
	}
	ret.RichTexts = &resultSet

	return &ret, nil
}
