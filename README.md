# demostore_api

## Environment variables

Besides the existing `DB_*`/`SERVER_PORT` variables, set `JWT_SECRET` to a strong random value in production (used to sign and verify access tokens). If unset, a fixed development default is used — do not rely on that default outside local development.

## Payments (Stripe)

Set these variables to enable payments (the server starts without them, logging a warning, but payment calls will fail):

- `STRIPE_SECRET_KEY` — secret API key (`sk_test_...` in test mode).
- `STRIPE_WEBHOOK_SECRET` — signing secret of the webhook endpoint (`whsec_...`).
- `PENDING_ORDER_TTL` — how long a new order may stay unpaid before it is cancelled and its stock released (Go duration, default `30m`).
- `BOLETO_EXPIRES_AFTER_DAYS` — calendar days until an issued boleto expires (0–60, default `3`).

Flow:

1. `POST /api/v1/orders` — checkout creates a `pending` order (amounts in cents, total between R$ 0,50 and R$ 999.999,99) with an `expires_at`.
2. `POST /api/v1/orders/{id}/payment` — returns the PaymentIntent `client_secret`; the frontend confirms the payment with Stripe.js / Elements. Calling it again returns the same intent.
3. Stripe calls `POST /api/v1/webhooks/stripe`; on `payment_intent.succeeded` the order becomes `paid`. The order is never marked paid from the client side.

Payment methods are offered by order total, following Stripe's limits: card always, Pix from R$ 0,50 to R$ 3.000, boleto from R$ 5,00 to R$ 49.999,99. Enable Pix and boleto in the Stripe dashboard.

| | Card | Pix | Boleto |
|---|---|---|---|
| Expiration | order `expires_at` | QR code expires with the order | order kept until the boleto expires + 5 days (Stripe confirms payments one business day later) |
| Cancel before paying | yes | yes | no — Stripe only allows it after the boleto expires |
| Refunds | full and partial | full and partial, up to 90 days | **not supported by Stripe** — refund the customer outside of it |

A background job cancels expired pending orders every minute, cancelling their PaymentIntent and restocking. Orders created before `expires_at` existed never expire automatically.

Cancelling a pending order cancels its PaymentIntent; an admin cancelling a paid order refunds the remaining amount. A payment that succeeds for an order cancelled in the meantime is refunded automatically (for boleto this is logged as `MANUAL REFUND REQUIRED`).

Admins refund items of a paid order with `POST /api/v1/admin/orders/{id}/refunds`:

```json
{ "items": [{ "product_id": "…", "quantity": 1 }], "reason": "damaged" }
```

The amount comes from the prices recorded at checkout, and the items go back to stock once Stripe confirms the refund. `GET /api/v1/admin/orders/{id}/refunds` lists them. Refunds made directly in the Stripe dashboard are not tracked by the API.

In the Stripe dashboard, subscribe the webhook endpoint to `payment_intent.succeeded`, `payment_intent.payment_failed`, `payment_intent.canceled`, `payment_intent.requires_action`, `refund.created`, `refund.updated` and `refund.failed`.

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
