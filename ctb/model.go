package ctb

import (
	"encoding/xml"
)

const (
	CtNodeSyntaxRichText  = "custom-colors"
	CtNodeSyntaxPlainText = "plain-text"
	CtDocElementText      = "text"
	CtDocElementCodeBox   = "code-box"
	CtDocElementTable     = "grid"
	CtDocElementPng       = "image-png"
	CtDocElementEmbFile   = "image-embfile"
	CtDocElementAnchor    = "image-anchor"
)

// CtNode Node data just without content
type CtNode struct {
	Id            int32  `json:"id"`            //
	Name          string `json:"name"`          //
	IsBold        bool   `json:"isBold"`        //
	IsCustomColor bool   `json:"isCustomColor"` //
	Color         uint32 `json:"color"`         // 节点标题的颜色（如果自定义的话），使用低3字节表示RGB
	IsReadOnly    bool   `json:"isReadOnly"`    //
	Icon          uint32 `json:"icon"`          // 图标的ID，cherrytree有一批编号过的图标
	IsRichText    bool   `json:"isRichText"`    //
	Syntax        string `json:"syntax"`        // 节点类型，custom-colors表示富文本（判断富文本应该通过IsRichText成员），plain-text表示纯文本，其它表示代码页对应的语言
	HasChildren   bool   `json:"hasChildren"`   //
}

func NewCtNode(pt *ptNodeMeta, hasChildren bool) *CtNode {
	return &CtNode{
		Id:            pt.NodeId,
		Name:          pt.Name,
		IsRichText:    pt.IsRichtxt&0b0001 != 0,  // 第1位表示是否为富文本节点
		IsBold:        pt.IsRichtxt&0b0010 != 0,  // 第2位表示是否加粗标题
		IsCustomColor: pt.IsRichtxt&0b0100 != 0,  // 第3位表示是否自定义标题颜色
		Color:         uint32(pt.IsRichtxt >> 3), // 第4位起3个字节表示RGB
		IsReadOnly:    pt.IsRo&0b1 != 0,          // 第1位表示是否只读
		Icon:          uint32(pt.IsRo >> 1),      // 其余位表示图标ID
		Syntax:        pt.Syntax,
		HasChildren:   hasChildren,
	}
}

type CtNodeContent struct {
	Id int32 `json:"id"`

	IsRichText bool `json:"isRichText"` // 富文本或代码页（包含纯文本）
	// 富文本的内容
	RichTexts *[][]interface{} `json:"richTexts,omitempty"`

	// 代码页语言
	Language string `json:"language,omitempty"`
	// 代码页内容
	Code string `json:"code,omitempty"`

	CreateTime int32 `json:"createTime"`
	UpdateTime int32 `json:"updateTime"`
}

// CtAnchoredWidget 附件类元素
type CtAnchoredWidget interface {
	GetOffset() int32
}

type _CtAnchoredWidgetMixin struct {
	Type          string `json:"type"`
	Offset        int32  `json:"offset,omitempty"`
	Justification string `json:"justification,omitempty"`
}

func (e _CtAnchoredWidgetMixin) GetOffset() int32 {
	return e.Offset
}
func (e _CtAnchoredWidgetMixin) GetType() string {
	return e.Type
}
func (e _CtAnchoredWidgetMixin) GetJustification() string {
	return e.Justification
}

// CtText 文本
type CtText struct {
	Type string `json:"type"`
	XmlRichText
}

func NewCtText(x *XmlRichText) *CtText {
	t := &CtText{
		Type:        CtDocElementText,
		XmlRichText: *x,
	}
	if t.Background != "" {
		runes := []rune(t.XmlRichText.Background)
		t.Background = string(append(append(runes[0:3], runes[5:7]...), runes[9:11]...)) // #eded33333b3b 不知道为什么是这种格式
	}
	if t.Foreground != "" {
		runes := []rune(t.XmlRichText.Foreground)
		t.Foreground = string(append(append(runes[0:3], runes[5:7]...), runes[9:11]...))
	}
	return t
}

// CtCodeBox 代码框
type CtCodeBox struct {
	_CtAnchoredWidgetMixin
	Code              string `json:"code"`
	Language          string `json:"language"`
	Width             int32  `json:"width"` // cherrytree 代码框是支持宽高设定的
	Height            int32  `json:"height"`
	IsWidthPixel      bool   `json:"isWidthPixel"` // 表明宽度的单位是像素（还是百分比）
	IsHighlightBraces bool   `json:"isHighlightBraces"`
	IsShowLineNumber  bool   `json:"isShowLineNumber"`
}

func NewCtCodeBox(t *tCodeBox) *CtCodeBox {
	return &CtCodeBox{
		_CtAnchoredWidgetMixin: _CtAnchoredWidgetMixin{
			Type:          CtDocElementCodeBox,
			Offset:        t.Offset,
			Justification: t.Justification,
		},
		Code:              t.Txt,
		Language:          t.Syntax,
		Width:             t.Width,
		Height:            t.Height,
		IsWidthPixel:      t.IsWidthPixel != 0,
		IsHighlightBraces: t.DoHighlightBraces != 0,
		IsShowLineNumber:  t.DoShowLineNumber != 0,
	}
}

// CtTable 表格
type CtTable struct {
	_CtAnchoredWidgetMixin
	Data        [][]string `json:"data"`
	MinColWidth int32      `json:"minColWidth"`
	MaxColWidth int32      `json:"maxColWidth"`
}

func NewCtTable(t *tGrid) *CtTable {
	u := XmlGrid{}
	err := xml.Unmarshal([]byte(t.Txt), &u)
	if err != nil {
		panic(err)
	}
	u.Rows = append(u.Rows[len(u.Rows)-1:], u.Rows[:len(u.Rows)-1]...)
	var data [][]string
	for _, r := range u.Rows {
		data = append(data, r.Cells)
	}
	return &CtTable{
		_CtAnchoredWidgetMixin: _CtAnchoredWidgetMixin{
			Type:          CtDocElementTable,
			Offset:        t.Offset,
			Justification: t.Justification,
		},
		Data:        data,
		MinColWidth: t.ColMin,
		MaxColWidth: t.ColMax,
	}
}

// CtPng 图片
type CtPng struct {
	_CtAnchoredWidgetMixin
	Data     []byte  `json:"data,omitempty"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	DiskPath *string `json:"diskPath,omitempty"`
}

func NewCtPng(i *tImage, width, height int, path *string) *CtPng {
	var data []byte
	if path == nil {
		data = i.Png
	}
	return &CtPng{
		_CtAnchoredWidgetMixin: _CtAnchoredWidgetMixin{
			Type:          CtDocElementPng,
			Offset:        i.Offset,
			Justification: i.Justification,
		},
		Data:     data,
		Width:    width,
		Height:   height,
		DiskPath: path,
	}
}

// CtEmbFile 附件
type CtEmbFile struct {
	_CtAnchoredWidgetMixin
	Data     []byte  `json:"data,omitempty"`
	Filename string  `json:"filename"`
	DiskPath *string `json:"diskPath,omitempty"`
}

func NewCtEmbFile(i *tImage, path *string) *CtEmbFile {
	var data []byte
	if path == nil {
		data = i.Png
	}
	return &CtEmbFile{
		_CtAnchoredWidgetMixin: _CtAnchoredWidgetMixin{
			Type:          CtDocElementEmbFile,
			Offset:        i.Offset,
			Justification: i.Justification,
		},
		Data:     data,
		Filename: i.Filename,
		DiskPath: path,
	}
}

// CtAnchor 锚
type CtAnchor struct {
	_CtAnchoredWidgetMixin
	Name string `json:"name"`
}

func NewCtAnchor(i *tImage) *CtAnchor {
	return &CtAnchor{
		_CtAnchoredWidgetMixin: _CtAnchoredWidgetMixin{
			Type:          CtDocElementAnchor,
			Offset:        i.Offset,
			Justification: i.Justification,
		},
		Name: i.Anchor,
	}
}
