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

public class MysqlDbTestExtension implements BeforeEachCallback, AfterEachCallback {
    private final String migrationsPath;

    private final String user = envOrDefault("MYSQL_USER", "root");
    private final String pass = envOrDefault("MYSQL_ROOT_PASSWORD", "mysecretpassword");
    private final String host = envOrDefault("MYSQL_HOST", "127.0.0.1");
    private final String port = envOrDefault("MYSQL_PORT", "3306");
    private final String mainDb = envOrDefault("MYSQL_DATABASE", "dinotest");
    private final String testDb = "sqltest_mysql";

    public MysqlDbTestExtension(String migrationsPath) {
        this.migrationsPath = migrationsPath;
    }

    @Override
    public void beforeEach(ExtensionContext context) throws Exception {
        getConnection(mainDb).createStatement().execute("CREATE DATABASE " + testDb);
        for (String migration : readMigrations()) {
            getConnection().createStatement().execute(migration);
        }
    }

    @Override
    public void afterEach(ExtensionContext context) throws Exception {
        getConnection(mainDb).createStatement().execute("DROP DATABASE " + testDb);
    }

    public Connection getConnection() throws SQLException {
        return getConnection(testDb);
    }

    private Connection getConnection(String db) throws SQLException {
        String url = "jdbc:mysql://" + host + ":" + port + "/" + db
                + "?user=" + user + "&password=" + pass + "&allowMultiQueries=true";
        return DriverManager.getConnection(url);
    }

    private List<String> readMigrations() throws IOException {
        Path path = Paths.get(migrationsPath);
        if (Files.isDirectory(path)) {
            try (Stream<Path> files = Files.list(path)) {
                return files
                        .filter(p -> p.toString().endsWith(".sql"))
                        .sorted()
                        .map(MysqlDbTestExtension::readString)
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
