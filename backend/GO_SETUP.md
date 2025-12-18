# Go 环境配置指南

## 问题：Go 安装后无法识别

如果 Go 已安装但无法识别，通常是 PATH 环境变量未配置。

## 快速修复

### 方法一：临时修复（当前终端会话）

```bash
export PATH=/usr/local/go/bin:$PATH
go version  # 验证是否生效
```

### 方法二：永久修复（推荐）

根据你的 shell 类型，将以下内容添加到对应的配置文件中：

#### Zsh (macOS 默认)

```bash
# 编辑 ~/.zshrc
nano ~/.zshrc

# 添加以下内容
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# 保存后重新加载
source ~/.zshrc
```

#### Bash

```bash
# 编辑 ~/.bash_profile 或 ~/.bashrc
nano ~/.bash_profile

# 添加以下内容
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# 保存后重新加载
source ~/.bash_profile
```

## 验证安装

```bash
# 检查 Go 版本
go version

# 检查 Go 环境变量
go env GOROOT
go env GOPATH
```

## 常见安装位置

- macOS (Homebrew): `/opt/homebrew/bin/go`
- 官方安装包: `/usr/local/go/bin/go`
- 用户安装: `$HOME/go/bin/go`

## 如果 Go 未安装

### macOS 使用 Homebrew

```bash
brew install go
```

### 从官网安装

1. 访问 https://go.dev/dl/
2. 下载适合 macOS 的安装包
3. 运行安装程序
4. 按照上面的方法配置 PATH

### 验证安装

```bash
go version
# 应该显示类似: go version go1.24.3 darwin/arm64
```

## 故障排查

### 问题：命令仍然找不到

1. **检查 Go 是否真的安装了**:
   ```bash
   ls -la /usr/local/go/bin/go
   ```

2. **检查 PATH 是否包含 Go**:
   ```bash
   echo $PATH | grep go
   ```

3. **重新打开终端**:
   - 关闭当前终端窗口
   - 打开新的终端窗口
   - 再次尝试 `go version`

4. **检查配置文件是否正确加载**:
   ```bash
   # 对于 zsh
   source ~/.zshrc
   
   # 对于 bash
   source ~/.bash_profile
   ```

### 问题：版本不匹配

PandaWiki 需要 Go 1.24.3 或更高版本：

```bash
go version
# 如果版本低于 1.24.3，需要升级
```

## 开发环境变量

除了基本的 PATH 配置，还可以设置以下环境变量（可选）：

```bash
# Go 工作空间
export GOPATH=$HOME/go

# Go 模块代理（可选，加速下载）
export GOPROXY=https://goproxy.cn,direct

# Go 私有模块（如果有）
export GOPRIVATE=*.private.com
```

将这些添加到你的 `~/.zshrc` 或 `~/.bash_profile` 中。

