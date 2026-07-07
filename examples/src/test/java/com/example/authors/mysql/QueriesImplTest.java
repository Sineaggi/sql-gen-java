package com.example.authors.mysql;

import com.example.dbtest.MysqlDbTestExtension;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

public class QueriesImplTest {

    @RegisterExtension
    static final MysqlDbTestExtension dbtest =
            new MysqlDbTestExtension("src/main/resources/authors/mysql/schema.sql");

    @Test
    void testCreateAuthor() throws Exception {
        Queries db = new QueriesImpl(dbtest.getConnection());

        List<Author> initialAuthors = db.listAuthors();
        assertTrue(initialAuthors.isEmpty());

        String name = "Brian Kernighan";
        String bio = "Co-author of The C Programming Language and The Go Programming Language";
        long id = db.createAuthor(name, bio);
        assertEquals(1L, id);
        Author expectedAuthor = new Author(id, name, bio);

        Author fetchedAuthor = db.getAuthor(id);
        assertEquals(expectedAuthor, fetchedAuthor);

        List<Author> listedAuthors = db.listAuthors();
        assertEquals(1, listedAuthors.size());
        assertEquals(expectedAuthor, listedAuthors.get(0));

        long id2 = db.createAuthor(name, bio);
        assertEquals(2L, id2);
    }

    @Test
    void testNull() throws Exception {
        Queries db = new QueriesImpl(dbtest.getConnection());

        List<Author> initialAuthors = db.listAuthors();
        assertTrue(initialAuthors.isEmpty());

        String name = "Brian Kernighan";
        String bio = null;
        long id = db.createAuthor(name, bio);
        Author expectedAuthor = new Author(id, name, bio);

        Author fetchedAuthor = db.getAuthor(id);
        assertEquals(expectedAuthor, fetchedAuthor);

        List<Author> listedAuthors = db.listAuthors();
        assertEquals(1, listedAuthors.size());
        assertEquals(expectedAuthor, listedAuthors.get(0));
    }
}
