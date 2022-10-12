package ctb

import (
	"fmt"
)

type tNode struct {
	NodeId     int32
	Name       string
	Txt        string
	Syntax     string
	Tags       string
	IsRo       int32 // readonly info and icon id
	IsRichtxt  int32
	HasCodebox int32
	HasTable   int32
	HasImage   int32
	Level      int32
	TsCreation int32
	TsLastsave int32
}

func (n tNode) TableName() string {
	return "node"
}

func (n tNode) String() string {
	return fmt.Sprintf(
		"NodeId=%d, Name=%q, Syntax=%q, IsRo=%032b, IsRichtxt=%032b, CreateTime=%d, UpdateTime=%d",
		n.NodeId, n.Name, n.Syntax, n.IsRo, n.IsRichtxt, n.TsCreation, n.TsLastsave,
	)
}

type ptNodeMeta struct {
	NodeId    int32
	Name      string
	Syntax    string
	IsRo      int32
	IsRichtxt int32
	Level     int32
}

type ptNodeContent struct {
	NodeId     int32
	Txt        string
	Syntax     string
	IsRichtxt  int32
	HasCodebox int32
	HasTable   int32
	HasImage   int32
	TsCreation int32
	TsLastsave int32
}

type tChildren struct {
	NodeId   int32
	FatherId int32
	Sequence int32
}

func (c tChildren) TableName() string {
	return "children"
}

type tImage struct {
	NodeId        int32
	Offset        int32
	Justification string
	Anchor        string
	Png           []byte
	Filename      string
	Link          string
	Time          int32
}

func (i tImage) TableName() string {
	return "image"
}

type tGrid struct {
	NodeId        int32
	Offset        int32
	Justification string
	Txt           string
	ColMin        int32
	ColMax        int32
}

func (g tGrid) TableName() string {
	return "grid"
}

type tCodeBox struct {
	NodeId            int32
	Offset            int32
	Justification     string
	Txt               string
	Syntax            string
	Width             int32
	Height            int32
	IsWidthPixel      int32 `gorm:"column:is_width_pix"` // 宽度单位为像素，否则为百分比
	DoHighlightBraces int32 `gorm:"column:do_highl_bra"`
	DoShowLineNumber  int32 `gorm:"column:do_show_linenum"`
}

func (c tCodeBox) TableName() string {
	return "codebox"
}
