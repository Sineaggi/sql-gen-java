package com.example.ondeck.mysql;

import com.example.dbtest.MysqlDbTestExtension;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.RegisterExtension;

import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class QueriesImplTest {

    @RegisterExtension
    static final MysqlDbTestExtension dbtest =
            new MysqlDbTestExtension("src/main/resources/ondeck/mysql/schema");

    @Test
    void testQueries() throws Exception {
        Queries q = new QueriesImpl(dbtest.getConnection());
        q.createCity("San Francisco", "san-francisco");
        City city = q.listCities().get(0);
        long venueId = q.createVenue(
                "the-fillmore",
                "The Fillmore",
                city.slug(),
                "spotify=uri",
                VenueStatus.OPEN,
                String.join(",", VenueStatus.OPEN.name(), VenueStatus.CLOSED.name()),
                String.join(",", "rock", "punk"));
        Venue venue = q.getVenue("the-fillmore", city.slug());
        assertEquals(venueId, venue.id());

        assertEquals(city, q.getCity(city.slug()));
        assertEquals(List.of(new VenueCountByCityRow(city.slug(), 1)), q.venueCountByCity());
        assertEquals(List.of(city), q.listCities());
        assertEquals(List.of(venue), q.listVenues(city.slug()));

        q.updateCityName("SF", city.slug());
        q.updateVenueName("Fillmore", venue.slug());
        Venue fresh = q.getVenue(venue.slug(), city.slug());
        assertEquals("Fillmore", fresh.name());

        q.deleteVenue(venue.slug(), venue.slug());
    }
}
