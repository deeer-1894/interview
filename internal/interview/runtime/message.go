package runtime

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func systemMessage(content string) adk.Message {
	return schema.SystemMessage(content)
}
