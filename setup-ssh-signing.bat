@echo off
REM GitHub SSH Signing Setup Script for Windows
REM Run as Administrator if needed for .ssh dir

echo === GitHub SSH Signing Setup ===
echo.

set /p name="Enter your full name (for git): "
set /p email="Enter your GitHub email: "

echo Creating .ssh directory...
mkdir "%USERPROFILE%\.ssh" 2^>nul

echo Generating SSH signing key...
ssh-keygen -t ed25519 -C "%email%" -f "%USERPROFILE%\.ssh\id_ed25519_signing" -N ""

echo Configuring git...
git config --global user.name "%name%"
git config --global user.email "%email%"
git config --global gpg.format ssh
git config --global user.signingkey "%USERPROFILE%\.ssh\id_ed25519_signing"
git config --global commit.gpgsign true

echo.
echo === NEXT STEPS ===
echo 1. Auth gh CLI with signing scope:
echo    gh auth refresh -h github.com -s admin:ssh_signing_key
echo.
echo 2. Add key to GitHub:
echo    gh ssh-key add "%USERPROFILE%\.ssh\id_ed25519_signing.pub" --type signing --title "Signing key (Windows)"
echo.
echo OR manually copy this key to https://github.com/settings/keys ^> New GPG key ^> Key type: Authentication key:
type "%USERPROFILE%\.ssh\id_ed25519_signing.pub"
echo.
echo 3. Test signing:
echo    echo test ^| git commit-tree HEAD^^^{tree} -S
echo.
echo Setup complete! All future git commits will be signed and verified on GitHub.
pause