// Package lark
// @Author: yinwei
// @File: lark_model
// @Version: 1.0.0
// @Date: 2024/8/28 09:56

package lark

type LarkCardLayout string

// card布局，详见 https://open.larksuite.com/document/common-capabilities/message-card/getting-started/card-structure/card-content
const (
	// 多列布局，可以横向排布多个列容器，在列内纵向自由组合图文内容，解决多列内容对齐问题，并实现了灵活的图文混排。详情参见多列布局。
	CARD_LAYOUT_COLUMN_SET LarkCardLayout = "column_set"
	// 内容模块，以格式化的文本为主体，支持混合图片、交互组件的富文本内容。详情参见内容模块。
	CARD_LAYOUT_DIV LarkCardLayout = "div"
	// 使用 Markdown 标签构造富文本内容。详情参见 Markdown。
	CARD_LAYOUT_MARKDOWN LarkCardLayout = "markdown"
	// 模块之间的分割线。详情参见分割线。
	CARD_LAYOUT_HR LarkCardLayout = "hr"
	//用于展示图片的组件。详情参见图片。
	CARD_LAYOUT_IMG LarkCardLayout = "img"
	//备注组件，用于展示卡片内的次要信息。详情参见备注。
	CARD_LAYOUT_NOTE LarkCardLayout = "note"
	//交互模块。使用交互组件可以实现消息卡片与用户之间的信息交互。详情参见交互模块。
	CARD_LAYOUT_ACTIONS = "actions"
)

type LarkCardTileColor string

const (
	CARD_TITLE_COLOR_BLUE    LarkCardTileColor = "blue"
	CARD_TITLE_COLOR_YELLOW  LarkCardTileColor = "yellow"
	CARD_TITLE_COLOR_RED     LarkCardTileColor = "red"
	CARD_TITLE_COLOR_GREY    LarkCardTileColor = "grey"
	CARD_TITLE_COLOR_DEFAULT LarkCardTileColor = "default"
)

// lark card 消息模板
type LarkCardMsg struct {
	Header   Header     `json:"header"`
	Elements []Elements `json:"elements"`
}

// 卡片的配置
type Config struct {
	// 是否允许转发卡片 默认为true
	EnableForward bool `json:"enable_forward"`
	// 是否为共享卡片 默认为false
	UpdateMulti bool `json:"update_multi"`
}

// 卡片头部，标题配置等
type Header struct {
	Title struct {
		//文本标识。固定取值：plain_text
		Tag string `json:"tag"`
		// 卡片主标题内容。
		Content string `json:"content"`
	} `json:"title"`
	// 标题主题颜色。可选值与示例效果参见下文的
	Template LarkCardTileColor `json:"template"`
}

// 卡片内容
type Elements struct {
	Tag LarkCardLayout `json:"tag"`

	Markdown

	Note

	Img

	Div

	CardColumnSet
}

type Markdown struct {
	// markdwon内容
	Content string `json:"content,omitempty"`
}

// 备注组件，用于展示卡片内的次要信息
type Note struct {
	Elements []struct {
		// 文本组件的数据结构，构成备注信息的文本内容。数据结构参见文本组件。
		//image 元素的数据结构，构成备注信息的小尺寸图片。数据结构参见 image 元素。
		Tag     string `json:"tag"`
		Content string `json:"content"`
	} `json:"elements,omitempty"`
}

type Img struct {
	ImgKey string `json:"img_key,omitempty"`
	Alt    struct {
		Tag     string `json:"tag"`
		Content string `json:"content"`
	} `json:"alt,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Preview bool   `json:"preview,omitempty"`
}

type Div struct {
	Text struct {
		Tag     string `json:"tag"`
		Content string `json:"content"`
	} `json:"text,omitempty"`
}

// column_set：多列布局，可以横向排布多个列容器，在列内纵向自由组合图文内容，解决多列内容对齐问题，并实现了灵活的图文混排。详情参见多列布局(https://open.larksuite.com/document/common-capabilities/message-card/message-cards-content/column-set)。
type CardColumnSet struct {
	FlexMode        string `json:"flex_mode,omitempty"`
	BackgroundStyle string `json:"background_style,omitempty"`
	Columns         []struct {
		Tag           string        `json:"tag"`
		Width         string        `json:"width"`
		Weight        int           `json:"weight"`
		VerticalAlign string        `json:"vertical_align"`
		Elements      []interface{} `json:"elements"`
	} `json:"columns,omitempty"`
}
