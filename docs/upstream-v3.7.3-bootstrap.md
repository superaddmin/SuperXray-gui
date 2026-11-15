# v3.7.3 首次账号初始化适配

来源：`MHSanaei/3x-ui` 提交 `1df8ab29f8cb1526be4e892ac0b03691fa3c6db9`。上游取消固定初始账号，并在服务首次启动时生成、记录随机凭据。本项目吸收“新库不能使用固定默认凭据登录”的修复目标，采用显式 CLI 初始化；不会生成或记录初始明文密码。

## 行为与兼容性

- 新建数据库及反复初始化均保持 `users` 空表，登录鉴权拒绝 `admin/admin`，也没有免鉴权的账号创建接口。
- `setting -show true` 对无用户的新库输出 `hasDefaultCredential: true`，既有 `install.sh` 会按原流程设置账号；已配置账号输出 `false`，历史默认账号仍输出 `true`。
- 直接运行二进制或 Docker 的新安装，需要先通过面板二进制的 `setting` 命令设置账号。服务日志仅提示初始化方式，不包含账号、密码或密码哈希。
- 升级已有数据库、准备导入候选库和重启不会旋转或删除已有账号。历史默认账号保持原值并提示需要设置，管理员应主动修改。
- 不新增表、字段或远程初始化入口；密码继续通过既有 `UserService.UpdateFirstUser` 写入 bcrypt 哈希。

## 首次配置

Linux 安装器会继续自动完成配置。手工部署时，在服务使用的数据库目录中执行实际面板二进制；`/usr/bin/x-ui` 可能是菜单脚本，不应把它当作二进制路径：

```bash
/usr/local/x-ui/x-ui setting -username '<用户名>' -password '<强密码>'
```

Docker 已启动但尚未配置账号时，使用同一容器内的数据库与二进制：

```bash
docker exec <容器名> /app/x-ui setting -username '<用户名>' -password '<强密码>'
```

自定义数据库目录须使用与服务相同的 `XUI_DB_FOLDER`。命令示例只含占位符；不要把真实凭据提交到仓库或自动化日志。设置完成后即可登录，无需通过日志寻找随机密码。

## 回归证据

先在原实现执行 `go test ./database . -run 'TestInitDBDoesNotSeedPanelCredentials|TestSettingCLI' -count=1`，数据库与 CLI 两项均因新库自动创建用户而失败，确认复现。

适配后执行 `go test ./database ./web/service . -count=1` 全部通过，覆盖新库重复初始化、安装器检测、显式设置账号、拒绝固定默认登录、账号持久化、既有账号保持与候选导入库准备；`go vet ./database .`、秘密扫描及格式/编码检查通过。代码落点为 `database/db.go`、`main.go`，回归见 `database/db_bootstrap_test.go`、`database/db_migration_test.go`、`main_bootstrap_test.go`。数据库仓库及迁移回滚的两处旧测试改为显式创建账号 fixture，不再依赖默认用户。
