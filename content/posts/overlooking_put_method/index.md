---
title: "You Might Be Overlooking the PUT Method in your REST API"
pubDate: 2023-08-08
Categories: ["System Design", "Go"]
Tags: ["System Design", "REST", "Backend", "Web Dev", "Learning", "Go"]
cover: "gallery/put_rest_api_cover.jpg"
---

So, you’re building your first or maybe your hundredth REST API, or perhaps you’re following along with a tutorial, and you’re about to implement the PUT HTTP method for an endpoint. To demonstrate a simple workflow for implementing a PUT endpoint,
let’s take a look at the following code snippets. In this article, I will be using Golang, but you can follow along with the language of your choice. Consider the following schema for a Pokemons table.

```sql
CREATE TABLE pokemons(
    id bigserial PRIMARY KEY,
    created_at TIMESTAMP(0) with time zone NOT NULL DEFAULT NO
    atk INTEGER NOT NULL,
    def INTEGER NOT NULL
);
```

Now, for the server logic, we will first invoke our GET method to retrieve the Pokémon that the user wants to update, like so.

```go
type PokemonModel struct {
 DB *sql.DB
}

func (m PokemonModel) Get(id int64) (*Pokemon, error) {

 if id < 1 {
  return nil, ErrRecordNotFound
 }

 query := `
        SELECT id, created_at, name, region, atk, def
        FROM pokemons
        WHERE id=$1
    `

 var pokemon Pokemon

 // Context to cancel our query, if it takes more than 3 seconds
 ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
 defer cancel()

 // Query the database and read the record into our pokemon struct
 err := m.DB.QueryRowContext(ctx, query, id).Scan(&pokemon.ID, &pokemon.CreatedAt,
           &pokemon.Name, &pokemon.Region, &pokemon.Atk, &pokemon.Def)

 if err != nil {
  switch {
  case errors.Is(err, sql.ErrNoRows):
   return nil, ErrRecordNotFound
  default:
   return nil, err
  }
 }

 return &pokemon, nil
}
```

Next, we create an update method that will be invoked with the Pokémon retrieved from the GET method to finalize the update.

```go
func (m PokemonModel) Update(pokemon *Pokemon) error {
 // update SQL query
 query := `
        UPDATE pokemons
        SET name=$1, region=$2, atk=$3, def=$4
        WHERE id=$5
        RETURNING id
    `

 // The values of our placeholder parameters
 args := []interface{}{pokemon.Name, pokemon.Region, pokemon.Atk,
                       pokemon.Def, pokemon.ID}

 // Context to cancel our query, if it takes more than 3 seconds
 ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
 defer cancel()

 // return the result of the update query (nil if no err, otherwise err)
 return m.DB.QueryRowContext(ctx, query, args...).Scan(&pokemon.ID)

}
```

Nice! We're done, right? Surely, there's no problem with this simple update. Well, what if I told you that you are about to fall into a **data race**[^1] - one of the most common and often overlooked problems when
performing updates on a mutable shared resource. Our Pokémon record is a shared resource that can be accessed by two or more threads simultaneously. Therefore, the order of operations,
as it stands, is dependent on the exact order in which the scheduler executes instructions.

In the above diagram, Alice makes a GET request (**api/pokemons/376**) and obtains the record of the Pokémon with id=376. Similarly, Bob also makes the same GET request.
At this point in time, both Alice and Bob have the Pokémon record represented by the green box, and a data race is about to occur when they both attempt to modify the original green record by sending a PUT request at approximately the same time.

In this scenario, Bob wants to change the **def field to 400**, while Alice wants to change the **atk field to 330**. However, as illustrated, only Alice's request is processed, resulting in the final green box representing only Alice's update.
Ideally, the record should have contained both updates instead of just Alice's. However, due to the requests being processed at around the same time, the scheduler has decided to apply Alice's update after Bob's, making Bob's request a victim of the data race.

[^1]: [Race condition](https://en.wikipedia.org/wiki/Race_condition)

## How to Deal With the Data Race?

There are multiple ways to address a data race, but the two main approaches are pessimistic locking and optimistic locking[^2]. In our case, we will utilize optimistic locking to handle the data race.
To protect our Pokémon record from a data race condition using optimistic locking, the following changes need to be implemented.

[^2]: [Optimistic vs Pessimistic Locking](https://stackoverflow.com/questions/129329/optimistic-vs-pessimistic-locking)

### Add a Version Column to the Pokemons Table

The first step to address the data race using optimistic locking, is to add a version column to our pokemons table. The version number will be defaulted to 1, as shown below:

```sql
ALTER TABLE pokemons
ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
```

### Include the Version Field in the Queries

Now, we need to include the version field in our queries. For the GET method will include version to the list of fields to be returned from the query.

```go
query := `
        SELECT id, created_at, name, region, atk, def, version
        FROM pokemons
        WHERE id=$1
    `
// Pass version as well to be stored in the struct (&pokemon.Version)
 err := m.DB.QueryRowContext(ctx, query, id).Scan(&pokemon.ID, &pokemon.CreatedAt,
 &pokemon.Name, &pokemon.Region, &pokemon.Atk, &pokemon.Def, &pokemon.Version)
```

The Update method will be slightly different. When making an update, we need to ensure that we increment the version number by 1, while simultaneously checking that the version number has not changed since we sent the GET request.

By increasing the version number by 1, we guarantee a unique, monotonically increasing column that can be checked against any other simultaneous queries. Including the version number in the conditional check ensures that we don't make changes to a record that was modified during the transaction.

The full code for the Update method now looks like this.

```go
// Custom error to notify of a data race
var ErrEditConflict = errors.new("edit conflict")

func (m PokemonModel) Update(pokemon *Pokemon) error {

 query := `
        UPDATE pokemons
        SET name=$1, region=$2, atk=$3, def=$4, version=version+1
        WHERE id=$5 AND version=$6
        RETURNING version
    `

 args := []interface{}{pokemon.Name, pokemon.Region, pokemon.Atk,
                       pokemon.Def, pokemon.ID, pokemon.Version}

 ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
 defer cancel()

 err := m.DB.QueryRowContext(ctx, query, args...).Scan(&pokemon.Version)

 if err != nil {
  switch {
  case errors.Is(err, sql.ErrNoRows):
   return ErrEditConflict
  default:
   return err
  }
 }
 return nil

}
```

Now that we have handled the data race, one possible follow-up is to make use of a custom error, such as **ErrEditConflict**, to send this event through an event queue and retry the failed query in a background worker. Regardless of the specific way you choose to deal with the conflict, the important thing is that you have taken action to solve the data race, and that is commendable.

Even though this is a simple example and the conflict may not seem harmful, it's crucial to recognize that such errors can lead to devastating consequences. For instance, when dealing with updates to an account's balance. Nevertheless, I hope this article has been helpful to you and has made you aware of the potential errors that can arise when implementing the PUT HTTP method in REST APIs.

## Resources

- [Let's Go Further By Alex Edwards. Chapter 6](https://lets-go-further.alexedwards.net/)
