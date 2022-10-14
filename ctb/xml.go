package ctb

import "encoding/xml"

type XmlDocument struct {
	XMLName   xml.Name      `xml:"node"`
	RichTexts []XmlRichText `xml:"rich_text" json:"richTexts"`
}

type XmlRichText struct {
	Foreground string `xml:"foreground,attr,omitempty" json:"foreground,omitempty"`
	Background string `xml:"background,attr,omitempty" json:"background,omitempty"`
	// 加粗 heavy
	Weight string `xml:"weight,attr,omitempty" json:"weight,omitempty"`

	Style string `xml:"style,attr,omitempty" json:"style,omitempty"`
	// 下划线 single
	Underline string `xml:"underline,attr,omitempty" json:"underline,omitempty"`
	// 删除线 true
	Strikethrough string `xml:"strikethrough,attr,omitempty" json:"strikethrough,omitempty"`
	// 标题、上下标 h1-h6 | sup | sub
	Scale string `xml:"scale,attr,omitempty" json:"scale,omitempty"`
	// 等宽 monospace
	Family string `xml:"family,attr,omitempty" json:"family,omitempty"`
	// 超链接 file folder webs
	Link          string `xml:"link,attr,omitempty" json:"link,omitempty"`
	Justification string `xml:"justification,attr,omitempty" json:"justification,omitempty"`
	Indent        int32  `xml:"indent,attr,omitempty" json:"indent,omitempty"`
	Text          string `xml:",chardata" json:"text"`
}

type XmlGrid struct {
	XMLName xml.Name `xml:"table"`
	Rows    []struct {
		XMLName xml.Name `xml:"row"`
		Cells   []string `xml:"cell"`
	} `xml:"row"`
}
