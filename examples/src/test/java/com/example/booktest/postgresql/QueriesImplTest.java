package com.example.booktest.postgresql;

import com.example.dbtest.PostgresDbTestExtension;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import java.sql.Connection;
import java.time.OffsetDateTime;
import java.time.format.DateTimeFormatter;
import java.util.List;

public class QueriesImplTest {

    @RegisterExtension
    static final PostgresDbTestExtension dbtest =
            new PostgresDbTestExtension("src/main/resources/booktest/postgresql/schema.sql");

    @Test
    void testQueries() throws Exception {
        Connection conn = dbtest.getConnection();
        Queries db = new QueriesImpl(conn);
        Author author = db.createAuthor("Unknown Master");

        // Start a transaction
        conn.setAutoCommit(false);
        db.createBook(
                author.authorId(),
                "1",
                BookType.NONFICTION,
                "my book title",
                2016,
                OffsetDateTime.now(),
                List.of());

        Book b1 = db.createBook(
                author.authorId(),
                "2",
                BookType.NONFICTION,
                "the second book",
                2016,
                OffsetDateTime.now(),
                List.of("cool", "unique"));

        db.updateBook(
                "changed second title",
                List.of("cool", "disastor"),
                b1.bookId());

        Book b3 = db.createBook(
                author.authorId(),
                "3",
                BookType.NONFICTION,
                "the third book",
                2001,
                OffsetDateTime.now(),
                List.of("cool"));

        db.createBook(
                author.authorId(),
                "4",
                BookType.NONFICTION,
                "4th place finisher",
                2011,
                OffsetDateTime.now(),
                List.of("other"));

        // Commit transaction
        conn.commit();
        conn.setAutoCommit(true);

        // ISBN update fails because parameters are not in sequential order. After changing $N to ?, ordering is lost,
        // and the parameters are filled into the wrong slots.
        db.updateBookISBN(
                "never ever gonna finish, a quatrain",
                List.of("someother"),
                "NEW ISBN",
                b3.bookId());

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
        List<BooksByTagsRow> res = db.booksByTags(List.of("cool", "other", "someother"));
        for (BooksByTagsRow ab : res) {
            System.out.println("Book " + ab.bookId() + ": '" + ab.title() + "', Author: '" + ab.name()
                    + "', ISBN: '" + ab.isbn() + "' Tags: '" + ab.tags() + "'");
        }
    }
}
