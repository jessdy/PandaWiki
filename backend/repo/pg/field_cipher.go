package pg

import (
	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/utils"
)

func NewFieldCipher(config *config.Config) (*utils.FieldCipher, error) {
	return utils.NewFieldCipher(config.FieldEncryptionSecrets()...)
}
