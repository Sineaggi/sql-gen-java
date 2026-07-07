package com.example.ondeck.postgresql;

import com.example.dbtest.PostgresDbTestExtension;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class QueriesImplTest {

    @RegisterExtension
    static final PostgresDbTestExtension dbtest =
            new PostgresDbTestExtension("src/main/resources/ondeck/postgresql/schema");

    @Test
    void testQueries() throws Exception {
        Queries q = new QueriesImpl(dbtest.getConnection());
        City city = q.createCity("San Francisco", "san-francisco");
        Integer venueId = q.createVenue(
                "the-fillmore",
                "The Fillmore",
                city.slug(),
                "spotify=uri",
                Status.OPEN,
                List.of(Status.OPEN, Status.CLOSED),
                List.of("rock", "punk"));
        Venue venue = q.getVenue("the-fillmore", city.slug());
        assertEquals(venueId, venue.id());

        assertEquals(city, q.getCity(city.slug()));
        assertEquals(List.of(new VenueCountByCityRow(city.slug(), 1)), q.venueCountByCity());
        assertEquals(List.of(city), q.listCities());
        assertEquals(List.of(venue), q.listVenues(city.slug()));

        q.updateCityName("SF", city.slug());
        Integer id = q.updateVenueName("Fillmore", venue.slug());
        assertEquals(venue.id(), id);

        q.deleteVenue(venue.slug());
    }
}
