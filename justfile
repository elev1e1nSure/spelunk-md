set shell := ["sh", "-cu"]

bin := if os() == "windows" { "spelunk-md.exe" } else { "spelunk-md" }

# Показать доступные команды
default:
    @printf '\033[1m\033[38;5;116mspelunk-md\033[0m  \033[2mdev commands\033[0m\n'
    @printf '\033[38;5;183m◆\033[0m  \033[1mbuild\033[0m      \033[2m·\033[0m  compile {{bin}}\n'
    @printf '\033[38;5;183m◆\033[0m  \033[1minstall\033[0m    \033[2m·\033[0m  go install to \$GOPATH/bin\n'
    @printf '\033[38;5;183m◆\033[0m  \033[1mcheck\033[0m      \033[2m·\033[0m  vet + build verify\n'
    @printf '\033[38;5;183m◆\033[0m  \033[1mtidy\033[0m       \033[2m·\033[0m  go mod tidy\n'
    @printf '\033[38;5;183m◆\033[0m  \033[1mdry\033[0m        \033[2m·\033[0m  dry-run on current directory\n'
    @printf '\033[38;5;183m◆\033[0m  \033[1mclean\033[0m      \033[2m·\033[0m  remove built binary\n'

# Компилировать бинарник
build:
    @printf '\033[38;5;116m◆\033[0m  \033[1mBuilding\033[0m  \033[2m→ go build -o {{bin}} .\033[0m\n'
    @go build -o {{bin}} . && \
      printf '\033[38;5;114m✓\033[0m  \033[1m{{bin}}\033[0m  \033[2mready\033[0m\n'

# Установить в $GOPATH/bin
install:
    @printf '\033[38;5;116m◆\033[0m  \033[1mInstalling\033[0m  \033[2m→ go install .\033[0m\n'
    @go install . && \
      printf '\033[38;5;114m✓\033[0m  installed to \033[2m$(go env GOPATH)/bin\033[0m\n'

# go vet + компиляция
check:
    @printf '\033[38;5;116m◆\033[0m  \033[1mVet\033[0m  \033[2m→ go vet ./...\033[0m\n'
    @go vet ./... && printf '\033[38;5;114m✓\033[0m  vet passed\n'
    @printf '\033[38;5;116m◆\033[0m  \033[1mBuild\033[0m  \033[2m→ go build ./...\033[0m\n'
    @go build ./... && printf '\033[38;5;114m✓\033[0m  build passed\n'

# Привести go.mod и go.sum в порядок
tidy:
    @printf '\033[38;5;116m◆\033[0m  \033[1mTidying modules\033[0m  \033[2m→ go mod tidy\033[0m\n'
    @go mod tidy && \
      printf '\033[38;5;114m✓\033[0m  go.mod + go.sum updated\n'

# Dry-run на текущей директории
dry:
    @printf '\033[38;5;183m◆\033[0m  \033[1mDry run\033[0m  \033[2m→ go run . --dry-run\033[0m\n'
    @go run . --dry-run

# Удалить собранный бинарник
clean:
    @printf '\033[38;5;210m◆\033[0m  \033[1mCleaning\033[0m\n'
    @rm -f {{bin}} && \
      printf '\033[38;5;114m✓\033[0m  \033[2m{{bin}} removed\033[0m\n'
