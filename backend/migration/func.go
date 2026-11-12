package migration

import (
	"github.com/chaitin/panda-wiki/migration/fns"
)

type MigrationFuncs struct {
	NodeMigration          *fns.MigrationNodeVersion
	BotAuthMigration       *fns.MigrationCreateBotAuth
	StoredSecretsMigration *fns.MigrationProtectStoredSecrets
}

func (mf *MigrationFuncs) GetMigrationFuncs() []MigrationFunc {
	funcs := []MigrationFunc{}
	funcs = append(funcs, MigrationFunc{
		Name: mf.NodeMigration.Name,
		Fn:   mf.NodeMigration.Execute,
	})
	funcs = append(funcs, MigrationFunc{
		Name: mf.BotAuthMigration.Name,
		Fn:   mf.BotAuthMigration.Execute,
	})
	funcs = append(funcs, MigrationFunc{
		Name:   mf.StoredSecretsMigration.Name,
		Fn:     mf.StoredSecretsMigration.Execute,
		Always: true,
	})
	return funcs
}
