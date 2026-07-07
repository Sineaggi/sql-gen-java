package com.example.authors.postgresql;

import com.example.dbtest.PostgresDbTestExtension;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import java.util.List;

public class QueriesImplTest {

    @RegisterExtension
    static final PostgresDbTestExtension dbtest =
            new PostgresDbTestExtension("src/main/resources/authors/postgresql/schema.sql");

    @Test
    void testCreateAuthor() throws Exception {
        Queries db = new QueriesImpl(dbtest.getConnection());

        List<Author> initialAuthors = db.listAuthors();
        Assertions.assertTrue(initialAuthors.isEmpty());

        String name = "Brian Kernighan";
        String bio = "Co-author of The C Programming Language and The Go Programming Language";
        Author insertedAuthor = db.createAuthor(name, bio);
        Author expectedAuthor = new Author(insertedAuthor.id(), name, bio);
        Assertions.assertEquals(expectedAuthor, insertedAuthor);

        Author fetchedAuthor = db.getAuthor(insertedAuthor.id());
        Assertions.assertEquals(expectedAuthor, fetchedAuthor);

        List<Author> listedAuthors = db.listAuthors();
        Assertions.assertEquals(1, listedAuthors.size());
        Assertions.assertEquals(expectedAuthor, listedAuthors.get(0));
    }

    @Test
    void testNull() throws Exception {
        Queries db = new QueriesImpl(dbtest.getConnection());

        List<Author> initialAuthors = db.listAuthors();
        Assertions.assertTrue(initialAuthors.isEmpty());

        String name = "Brian Kernighan";
        String bio = null;
        Author insertedAuthor = db.createAuthor(name, bio);
        Author expectedAuthor = new Author(insertedAuthor.id(), name, bio);
        Assertions.assertEquals(expectedAuthor, insertedAuthor);

        Author fetchedAuthor = db.getAuthor(insertedAuthor.id());
        Assertions.assertEquals(expectedAuthor, fetchedAuthor);

        List<Author> listedAuthors = db.listAuthors();
        Assertions.assertEquals(1, listedAuthors.size());
        Assertions.assertEquals(expectedAuthor, listedAuthors.get(0));
    }
}
