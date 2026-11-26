# mydb

A relational database built from scratch in Go: slotted pages, a buffer pool, a SQL parser and an executor, with no external dependencies.

This is a learning project. The goal is to understand how a database works inside by building each layer myself, not to build a production database. The full write-up, with one chapter per concept and one per implementation, is at **[shwadfy-dev.vercel.app/learn](https://shwadfy-dev.vercel.app/learn/)**.

## Quick start

```bash
go build -o mydb ./src
./mydb
```

```
Welcome to mydb REPL. Type .exit to exit.
mydb> CREATE TABLE users (id INT, name VARCHAR);
Table created successfully
mydb> INSERT INTO users VALUES (1, 'muhammad');
Row inserted successfully
mydb> UPDATE users SET name = 'ali' WHERE id = 1;
1 row(s) updated
mydb> SELECT * FROM users WHERE id >= 1;
mydb> DELETE FROM users WHERE id = 1;
1 row(s) deleted
mydb> .exit
```

Statements end with `;` and can span several lines. Data is stored in `data.db` in the current directory.

> **Leave with `.exit`, not Ctrl+C.** Pages are flushed to disk on a clean shutdown only. There is no write-ahead log yet, so killing the process loses every change made since startup.

Run the tests:

```bash
go test ./...
```

## Supported SQL

```sql
CREATE TABLE <table> (<col> <type>, ...)
INSERT INTO <table> VALUES (<literal>, ...)
SELECT * FROM <table> [WHERE <col> <op> <literal>]
UPDATE <table> SET <col> = <literal> [, ...] [WHERE <col> <op> <literal>]
DELETE FROM <table> [WHERE <col> <op> <literal>]
```

- **Types:** `INT`, `BIGINT`, `VARCHAR`, `BOOLEAN`
- **Operators:** `=` `!=` `<` `>` `<=` `>=`
- **Not supported yet:** `NULL`, `AND`/`OR`, column lists in `INSERT` or `SELECT`, `DROP TABLE`, joins, indexes

## Architecture

```
SQL text
  → REPL ─→ Lexer ─→ Parser ─→ AST ─→ Executor
                                         │
                       Catalog ←─────────┤
                                         ↓
                    Tuple / Serializer ↔ TableHeap + Scanner
                                         ↓
                                    Buffer pool
                                         ↓
                                    Disk manager → data.db
```

Each layer only talks to the one directly below it.

| Layer | Package | What it does | Read more |
|---|---|---|---|
| Slotted page | `storageLayer` | 4 KB page with a slot directory, so a record keeps the same ID even when it moves inside the page | [Slotted pages](https://shwadfy-dev.vercel.app/learn/slotted-page-layout/), [first page](https://shwadfy-dev.vercel.app/learn/implementing-the-first-page/), [compaction](https://shwadfy-dev.vercel.app/learn/compaction/) |
| Disk manager | `storageLayer` | Reads and writes whole pages to `data.db`, and forces each write to reach the disk (`fsync`) | [Disk manager](https://shwadfy-dev.vercel.app/learn/disk-manager/) |
| Buffer pool | `storageLayer` | Page cache with LRU eviction and pin counts; remembers which pages changed | [v1](https://shwadfy-dev.vercel.app/learn/buffer-pool/), [v2](https://shwadfy-dev.vercel.app/learn/buffer-pool-v2/) |
| Table heap | `tableManager` | A table is a linked list of pages; records are addressed by `RecordID{PageId, SlotNumber}` | [Table heap](https://shwadfy-dev.vercel.app/learn/table-heap/) |
| Scanner | `tableManager` | Walks every live record, keeping only one page pinned at a time | [Scanner](https://shwadfy-dev.vercel.app/learn/scanner/) |
| Tuple and serializer | `tuple` | Typed values and schemas; a row is encoded as a fixed-size region plus variable-length data | Chapter coming soon |
| Catalog | `catalog` | Records every table's name, first page and schema. It is itself stored as a table heap at a fixed page, so tables survive a restart | — |
| Lexer and parser | `sql` | Turns SQL text into tokens, then into a syntax tree (AST). Values in the SQL (like `25` or `'ali'`) get their real type only in the executor, because only the executor knows each column's type | Chapter coming soon |
| Executor | `executor` | Runs an AST against the catalog and table heaps. `UPDATE` and `DELETE` collect the matching `RecordID`s first and change them afterwards, so a row that moves is never visited twice | — |
| REPL | `repl` | Reads statements up to `;`, runs them and prints the results | — |

## File hierarchy

```
src/
├── main.go                    Connects disk manager → buffer pool → catalog → executor → REPL
├── storageLayer/
│   ├── page.go                Slotted page: header, slot directory, insert/get/update/delete, compaction
│   ├── diskManager.go         Page-sized reads and writes to data.db
│   └── bufferPool.go          Page cache: fetch, pin/unpin, LRU eviction, flush
├── tableManager/
│   ├── tableHeap.go           A table as a chain of pages; insert/get/update/delete by RecordID
│   └── scanner.go             Iterates over a table's live records, one page at a time
├── tuple/
│   ├── type.go                Database type IDs (INT, BIGINT, VARCHAR, BOOLEAN)
│   ├── value.go               Typed values and comparison
│   ├── column.go              Column definitions
│   ├── schema.go              Ordered columns and lookup by name
│   ├── tuple.go               A row: values validated against a schema
│   ├── serializer.go          Encodes rows to bytes and back
│   └── schema_serializer.go   Encodes a schema so the catalog can store it
├── catalog/
│   └── catalog.go             Table registry, stored as a table heap
├── sql/
│   ├── lexer.go               SQL text → tokens
│   ├── ast.go                 Statement types: CreateTable, Insert, Select, Update, Delete
│   └── parser.go              Tokens → AST
├── executor/
│   └── executor.go            Runs statements; literal → typed value; WHERE evaluation
└── repl/
    └── repl.go                Interactive shell
```

Test files (`*_test.go`) sit next to the code they cover.

## Roadmap

- [x] Slotted page layout
- [x] Compaction
- [x] Disk manager
- [x] Buffer pool (LRU, pin counts, dirty pages)
- [x] Table heap
- [x] Scanner
- [x] Schema, tuples and types
- [x] Serializer
- [x] Catalog (tables survive restarts)
- [x] SQL lexer and parser
- [x] Executor: `CREATE TABLE`, `INSERT`, `SELECT`
- [x] `WHERE` with all comparison operators
- [x] REPL
- [x] Executor: `UPDATE`, `DELETE`
- [ ] Write-ahead log and crash recovery **(in progress)**
- [ ] Checkpoints
- [ ] B+tree indexes
- [ ] `AND` / `OR`, `NULL`
- [ ] Transactions
- [ ] MVCC

## Known limitations

- **Data can be lost on a crash:** see the Ctrl+C note above. The WAL fixes this.
- **Single user:** there is no concurrency control or locking.
- **No transactions:** a statement that fails halfway through leaves the rows it already changed.
- **Size cap:** page IDs are `uint16`, so the database file is limited to 65,536 pages × 4 KB = 256 MB.
- **Raw output:** `SELECT` prints rows as Go structs; there is no table formatting yet.
