# demostore_api

## Environment variables

Besides the existing `DB_*`/`SERVER_PORT` variables, set `JWT_SECRET` to a strong random value in production (used to sign and verify access tokens). If unset, a fixed development default is used — do not rely on that default outside local development.

## Payments (Stripe)

Set these variables to enable payments (the server starts without them, logging a warning, but payment calls will fail):

- `STRIPE_SECRET_KEY` — secret API key (`sk_test_...` in test mode).
- `STRIPE_WEBHOOK_SECRET` — signing secret of the webhook endpoint (`whsec_...`).

Flow:

1. `POST /api/v1/orders` — checkout creates a `pending` order (amounts in cents).
2. `POST /api/v1/orders/{id}/payment` — returns the PaymentIntent `client_secret`; the frontend confirms the payment with Stripe.js / Elements. Calling it again returns the same intent.
3. Stripe calls `POST /api/v1/webhooks/stripe`; on `payment_intent.succeeded` the order becomes `paid`. The order is never marked paid from the client side.

Cancelling a pending order cancels its PaymentIntent; an admin cancelling a paid order refunds it. A payment that succeeds for an order cancelled in the meantime is refunded automatically.

In the Stripe dashboard, subscribe the webhook endpoint to `payment_intent.succeeded`, `payment_intent.payment_failed` and `payment_intent.canceled`.

To test webhooks locally with the [Stripe CLI](https://docs.stripe.com/stripe-cli):

```sh
stripe listen --forward-to localhost:8080/api/v1/webhooks/stripe
# use the printed whsec_... as STRIPE_WEBHOOK_SECRET
```

## Promoting a user to admin

There is no self-service endpoint to become an admin. New users always register as `customer`. To promote an existing user, register normally first and then run directly against the database:

```sql
UPDATE users SET role = 'admin' WHERE email = 'someone@example.com';
```

The user needs to log in again afterwards to get a new token carrying the updated role.
