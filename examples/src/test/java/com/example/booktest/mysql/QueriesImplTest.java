package com.example.booktest.mysql;

import com.example.dbtest.MysqlDbTestExtension;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import java.sql.Connection;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.List;

public class QueriesImplTest {

    @RegisterExtension
    static final MysqlDbTestExtension dbtest =
            new MysqlDbTestExtension("src/main/resources/booktest/mysql/schema.sql");

    @Test
    void testQueries() throws Exception {
        Connection conn = dbtest.getConnection();
        Queries db = new QueriesImpl(conn);
        long authorId = db.createAuthor("Unknown Master");
        Author author = db.getAuthor((int) authorId);

        // Start a transaction
        conn.setAutoCommit(false);
        db.createBook(
                author.authorId(),
                "1",
                BooksBookType.NONFICTION,
                "my book title",
                2016,
                LocalDateTime.now(),
                "");

        long b1Id = db.createBook(
                author.authorId(),
                "2",
                BooksBookType.NONFICTION,
                "the second book",
                2016,
                LocalDateTime.now(),
                String.join(",", List.of("cool", "unique")));

        db.updateBook(
                "changed second title",
                String.join(",", List.of("cool", "disastor")),
                (int) b1Id);

        long b3Id = db.createBook(
                author.authorId(),
                "3",
                BooksBookType.NONFICTION,
                "the third book",
                2001,
                LocalDateTime.now(),
                String.join(",", List.of("cool")));

        db.createBook(
                author.authorId(),
                "4",
                BooksBookType.NONFICTION,
                "4th place finisher",
                2011,
                LocalDateTime.now(),
                String.join(",", List.of("other")));

        // Commit transaction
        conn.commit();
        conn.setAutoCommit(true);

        db.updateBookISBN(
                "never ever gonna finish, a quatrain",
                String.join(",", List.of("someother")),
                "NEW ISBN",
                (int) b3Id);

        List<Book> books0 = db.booksByTitleYear("my book title", 2016);

        DateTimeFormatter formatter = DateTimeFormatter.ISO_DATE_TIME;
        for (Book book : books0) {
            System.out.println("Book " + book.bookId() + " (" + book.bookType() + "): " + book.title()
                    + " available: " + book.available().format(formatter));
            Author author2 = db.getAuthor(book.authorId());
            System.out.println("Book " + book.bookId() + " author: " + author2.name());
        }

        // find a book with either "cool" or "other" tag
        System.out.println("---------\nTag search results:\n");
        List<BooksByTagsRow> res = db.booksByTags(String.join(",", List.of("cool", "other", "someother")));
        for (BooksByTagsRow ab : res) {
            System.out.println("Book " + ab.bookId() + ": '" + ab.title() + "', Author: '" + ab.name()
                    + "', ISBN: '" + ab.isbn() + "' Tags: '" + ab.tags() + "'");
        }
    }
}
