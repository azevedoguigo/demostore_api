# demostore_api

## Environment variables

Besides the existing `DB_*`/`SERVER_PORT` variables, set `JWT_SECRET` to a strong random value in production (used to sign and verify access tokens). If unset, a fixed development default is used — do not rely on that default outside local development.

## Promoting a user to admin

There is no self-service endpoint to become an admin. New users always register as `customer`. To promote an existing user, register normally first and then run directly against the database:

```sql
UPDATE users SET role = 'admin' WHERE email = 'someone@example.com';
```

The user needs to log in again afterwards to get a new token carrying the updated role.
