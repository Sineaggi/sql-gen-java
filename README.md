# sqlc-gen-java

A [sqlc](https://sqlc.dev) plugin that generates Java from your SQL schema and
queries. For each package it emits:

- one Java `record` per table model and per query row-result type
- a `Queries` interface
- a JDBC-backed `QueriesImpl`

It targets **Java 17** (records, text blocks) and, by default, annotates
nullable values with [JSpecify](https://jspecify.dev)'s `@Nullable`
(`org.jspecify.annotations.Nullable`) plus a `@NullMarked` `package-info.java`.

## Usage

```yaml
version: '2'
plugins:
- name: java
  wasm:
    url: https://downloads.sqlc.dev/plugin/sqlc-gen-java.wasm
    sha256: ""
sql:
- schema: src/main/resources/authors/postgresql/schema.sql
  queries: src/main/resources/authors/postgresql/query.sql
  engine: postgresql
  codegen:
  - out: src/main/java/com/example/authors/postgresql
    plugin: java
    options:
      package: com.example.authors.postgresql
```

### Options

| Option                      | Type    | Default | Description                                                                                          |
|-----------------------------|---------|---------|------------------------------------------------------------------------------------------------------|
| `package`                   | string  | `""`    | The Java package for the generated sources.                                                          |
| `emit_jspecify_annotations` | bool    | `true`  | Emit JSpecify `@Nullable`/`@NullMarked`. Set to `false` to drop the JSpecify dependency entirely.    |
| `emit_exact_table_names`    | bool    | `false` | Skip inflection (singularization) when deriving model names from table names.                        |

When `emit_jspecify_annotations` is enabled you need JSpecify on the compile
classpath (it is compile-only — CLASS retention, not required at runtime):

```groovy
compileOnly 'org.jspecify:jspecify:1.0.0'
```

## Building locally

```shell
make all            # builds bin/sqlc-gen-java and bin/sqlc-gen-java.wasm
```

To run against a local build, point `wasm.url` at the file (the `sha256` may be
omitted for a local `file://` plugin — sqlc will compute it and warn):

```yaml
plugins:
- name: java
  wasm:
    url: file://../bin/sqlc-gen-java.wasm
```

See [`examples/`](examples/) for a full multi-schema project with tests.
