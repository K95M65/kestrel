# Google account

This robot does not join an Apple ID or a Google Home. It can still **sign in as you** for mail and calendar — the same idea as signing in on a TV.

1. Device → Channels → **Sign in with Google**. A code and QR appear.
2. Open google.com/device on your phone. Enter the code.
3. Mail and calendar then use that account. House → Behaviors **Ask** stays on important actions: mail is a draft until you say send.

Skip is fine: Gmail still accepts an app password, Calendar still accepts a secret iCal address.

The TV OAuth client (Google Cloud, type **TVs and Limited Input**) is a one-time operator step. If Sign in with Google is grey, use skip, or expand **I have a Google Cloud TV client**.

**Apple.** Sign in with Apple needs a public `https://` return URL. On this desk, use Google. Developer fields are Advanced (`?debug=true`).

See [Claim this robot](claim) for the HomeKit-style setup code.
