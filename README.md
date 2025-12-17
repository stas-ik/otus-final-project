gomigrator - инструмент миграций для PostgreSQL с поддержкой миграций на SQL и Go.

Установка CLI:

```
go install migrator/cmd/gomigrator@latest
```

Использование как библиотеки:

```
import "migrator/pkg/migrator"
```

Команды:
- gomigrator create <name> - создать шаблон миграции (SQL по умолчанию)
- gomigrator up - применить все доступные миграции
- gomigrator down - откатить последнюю примененную миграцию
- gomigrator redo - откатить и снова применить последнюю миграцию
- gomigrator status - вывести таблицу статуса миграций
- gomigrator dbversion - показать последнюю примененную версию

Конфигурация: YAML файл + переменные окружения + флаги CLI.
Пример config.yaml:

```
dsn: ${DB_DSN}
path: ./migrations
kind: sql # sql|go
lock_key: 7243392
schema_table: schema_migrations
```

SQL миграции: один файл с разделителями:

```
-- +migrate Up
CREATE TABLE example(id int);

-- +migrate Down
DROP TABLE example;
```

Go миграции: регистрация функций в реестре с идентификатором, совпадающим с именем файла/миграции.

Лицензия: MIT
