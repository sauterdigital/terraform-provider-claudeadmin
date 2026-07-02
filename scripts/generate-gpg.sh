#!/usr/bin/env bash
# Helper para gerar a GPG key usada pelo goreleaser assinar as releases do
# terraform-provider-anthropic antes do upload no Terraform Registry.
#
# Uso: ./scripts/generate-gpg.sh <email> [<real-name>]
#
# Requisitos: gpg 2.x instalado (brew install gnupg).

set -euo pipefail

EMAIL="${1:-}"
REAL_NAME="${2:-Sauter Digital}"

if [ -z "$EMAIL" ]; then
  echo "Uso: $0 <email> [<real-name>]"
  echo "  Ex: $0 opscare@sauter.digital 'Sauter Digital'"
  exit 2
fi

TMPDIR="$(mktemp -d)"
BATCH_FILE="$TMPDIR/gpg-batch"

cat > "$BATCH_FILE" <<EOF
%echo Gerando GPG key para $REAL_NAME <$EMAIL>
Key-Type: RSA
Key-Length: 4096
Key-Usage: sign
Subkey-Type: RSA
Subkey-Length: 4096
Subkey-Usage: sign
Name-Real: $REAL_NAME
Name-Email: $EMAIL
Expire-Date: 2y
%ask-passphrase
%commit
%echo done
EOF

echo ">>> Gerando GPG key (2048 bytes, RSA 4096, expira em 2 anos)..."
echo ">>> Você vai ser pedida uma passphrase. GUARDE NO VAULT (1Password/Bitwarden)."
echo

gpg --batch --generate-key "$BATCH_FILE"

echo
echo ">>> Key gerada. Localizando fingerprint..."
FINGERPRINT=$(gpg --list-secret-keys --keyid-format=long --with-colons "$EMAIL" \
  | awk -F: '/^fpr:/ {print $10; exit}')

if [ -z "$FINGERPRINT" ]; then
  echo "ERRO: não consegui encontrar a fingerprint. Rode 'gpg --list-secret-keys' manualmente."
  exit 1
fi

echo ">>> Fingerprint: $FINGERPRINT"
echo

PRIVATE_OUT="$TMPDIR/anthropic_provider_gpg.private.asc"
PUBLIC_OUT="$TMPDIR/anthropic_provider_gpg.public.asc"

gpg --armor --export-secret-keys "$FINGERPRINT" > "$PRIVATE_OUT"
gpg --armor --export "$FINGERPRINT" > "$PUBLIC_OUT"

echo ">>> Chaves exportadas:"
echo "    Private (upload no GitHub Secrets GPG_PRIVATE_KEY): $PRIVATE_OUT"
echo "    Public  (upload no Terraform Registry):             $PUBLIC_OUT"
echo
echo ">>> Próximos passos:"
echo "  1. GitHub → Settings → Secrets and variables → Actions:"
echo "     - GPG_PRIVATE_KEY: cat $PRIVATE_OUT | pbcopy"
echo "     - PASSPHRASE: a passphrase que você digitou (não fica no arquivo)"
echo
echo "  2. Depois de configurar secrets, rode o workflow 'release' no GitHub Actions."
echo
echo "  3. Terraform Registry → Publish → upload $PUBLIC_OUT"
echo
echo ">>> IMPORTANTE: shred/rm o arquivo privado depois de copiar:"
echo "    shred -u $PRIVATE_OUT   # macOS: gshred -u ou apenas rm"
echo
echo ">>> Fingerprint (anota isso no cofre também):"
echo "    $FINGERPRINT"
