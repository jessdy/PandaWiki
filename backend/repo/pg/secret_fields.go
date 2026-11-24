package pg

import (
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/utils"
)

func protectField(cipher *utils.FieldCipher, value *string) error {
	if cipher == nil || value == nil || *value == "" || !cipher.NeedsReencrypt(*value) {
		return nil
	}
	plain, err := cipher.Decrypt(*value)
	if err != nil {
		return err
	}
	enc, err := cipher.Encrypt(plain)
	if err != nil {
		return err
	}
	*value = enc
	return nil
}

func revealField(cipher *utils.FieldCipher, value *string) error {
	if cipher == nil || value == nil || !utils.IsFieldEncrypted(*value) {
		return nil
	}
	plain, err := cipher.Decrypt(*value)
	if err != nil {
		return err
	}
	*value = plain
	return nil
}

func SealSecret(cipher *utils.FieldCipher, value *string) error {
	return protectField(cipher, value)
}

func SealAppSettings(cipher *utils.FieldCipher, settings *domain.AppSettings) error {
	if settings == nil {
		return nil
	}
	fields := []*string{
		&settings.DingTalkBotClientSecret,
		&settings.FeishuBotAppSecret,
		&settings.LarkBotSettings.AppSecret,
		&settings.LarkBotSettings.VerifyToken,
		&settings.LarkBotSettings.EncryptKey,
		&settings.WeChatAppToken,
		&settings.WeChatAppEncodingAESKey,
		&settings.WeChatAppSecret,
		&settings.WeChatServiceToken,
		&settings.WeChatServiceEncodingAESKey,
		&settings.WeChatServiceSecret,
		&settings.DiscordBotToken,
		&settings.WechatOfficialAccountAppSecret,
		&settings.WechatOfficialAccountToken,
		&settings.WechatOfficialAccountEncodingAESKey,
		&settings.WecomAIBotSettings.Token,
		&settings.WecomAIBotSettings.EncodingAESKey,
		&settings.OpenAIAPIBotSettings.SecretKey,
	}
	for _, field := range fields {
		if err := protectField(cipher, field); err != nil {
			return err
		}
	}
	return nil
}

func openAppSettings(cipher *utils.FieldCipher, settings *domain.AppSettings) error {
	if settings == nil {
		return nil
	}
	fields := []*string{
		&settings.DingTalkBotClientSecret,
		&settings.FeishuBotAppSecret,
		&settings.LarkBotSettings.AppSecret,
		&settings.LarkBotSettings.VerifyToken,
		&settings.LarkBotSettings.EncryptKey,
		&settings.WeChatAppToken,
		&settings.WeChatAppEncodingAESKey,
		&settings.WeChatAppSecret,
		&settings.WeChatServiceToken,
		&settings.WeChatServiceEncodingAESKey,
		&settings.WeChatServiceSecret,
		&settings.DiscordBotToken,
		&settings.WechatOfficialAccountAppSecret,
		&settings.WechatOfficialAccountToken,
		&settings.WechatOfficialAccountEncodingAESKey,
		&settings.WecomAIBotSettings.Token,
		&settings.WecomAIBotSettings.EncodingAESKey,
		&settings.OpenAIAPIBotSettings.SecretKey,
	}
	for _, field := range fields {
		if err := revealField(cipher, field); err != nil {
			return err
		}
	}
	return nil
}
