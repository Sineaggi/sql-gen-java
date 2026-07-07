package com.example.dbtest;

import org.junit.jupiter.api.extension.AfterEachCallback;
import org.junit.jupiter.api.extension.BeforeEachCallback;
import org.junit.jupiter.api.extension.ExtensionContext;

import java.io.IOException;
import java.io.UncheckedIOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.SQLException;
import java.util.List;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class PostgresDbTestExtension implements BeforeEachCallback, AfterEachCallback {
    public static final String schema = "dinosql_test";

    private final String migrationsPath;
    private final Connection schemaConn;
    private final String url;

    public PostgresDbTestExtension(String migrationsPath) {
        this.migrationsPath = migrationsPath;

        String user = envOrDefault("PG_USER", "postgres");
        String pass = envOrDefault("PG_PASSWORD", "mysecretpassword");
        String host = envOrDefault("PG_HOST", "127.0.0.1");
        String port = envOrDefault("PG_PORT", "5432");
        String db = envOrDefault("PG_DATABASE", "dinotest");
        this.url = "jdbc:postgresql://" + host + ":" + port + "/" + db
                + "?user=" + user + "&password=" + pass + "&sslmode=disable";

        try {
            this.schemaConn = DriverManager.getConnection(url);
        } catch (SQLException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public void beforeEach(ExtensionContext context) throws Exception {
        schemaConn.createStatement().execute("CREATE SCHEMA " + schema);
        for (String migration : readMigrations()) {
            getConnection().createStatement().execute(migration);
        }
    }

    @Override
    public void afterEach(ExtensionContext context) throws Exception {
        schemaConn.createStatement().execute("DROP SCHEMA " + schema + " CASCADE");
    }

    public Connection getConnection() throws SQLException {
        return DriverManager.getConnection(url + "&currentSchema=" + schema);
    }

    private List<String> readMigrations() throws IOException {
        Path path = Paths.get(migrationsPath);
        if (Files.isDirectory(path)) {
            try (Stream<Path> files = Files.list(path)) {
                return files
                        .filter(p -> p.toString().endsWith(".sql"))
                        .sorted()
                        .map(PostgresDbTestExtension::readString)
                        .collect(Collectors.toList());
            }
        }
        return List.of(Files.readString(path));
    }

    private static String readString(Path p) {
        try {
            return Files.readString(p);
        } catch (IOException e) {
            throw new UncheckedIOException(e);
        }
    }

    private static String envOrDefault(String key, String fallback) {
        String value = System.getenv(key);
        return value != null ? value : fallback;
    }
}
