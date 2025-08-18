package generator

import (
	"syncai/internal/model"
)

type OtherRulesGenerator struct{}

func (g OtherRulesGenerator) GenerateRules(metadata model.RulesMetadata, content []byte) []byte {
	return content
}
