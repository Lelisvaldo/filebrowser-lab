package web

import (
	"os"
	"strings"

	"github.com/gtsteffaniak/filebrowser/backend/internal/database/users"
)

// labPasskeyMFARequired diz se, depois de acertar a senha, a passkey cadastrada ainda é exigida como segundo fator.
// Padrão (lab): NÃO. A passkey é um jeito a mais, e mais cômodo, de entrar (biometria sem digitar usuário e senha), e não uma
// exigência extra: sem isso, cadastrar uma chave num aparelho travaria o login por senha dos aparelhos que não conseguem usar
// passkey (como o WebView do app Android). Para voltar ao comportamento do Quantum (senha + passkey), defina
// FILEBROWSER_LAB_PASSKEY_MFA=on no container. Quem tem verificação em duas etapas (TOTP) continua protegido por ela.
func labPasskeyMFARequired(u *users.User) bool {
	if !u.HasPasskeyMFA() {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("FILEBROWSER_LAB_PASSKEY_MFA")), "on")
}
