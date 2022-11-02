package config

type Params struct {
	MarkOnErrorFlag         bool
	MarkOnNotFoundFlag      bool
	BrokersUrls             string
	ConsumersNumber         int
	RelativePath            string
	Topic                   string
	AsCLI                   bool
}
