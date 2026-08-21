# Google account

This robot does not join an Apple ID or a Google Home. It can still **sign in as you** for mail and calendar.

1. In Google Cloud, create an OAuth client of type **TV and Limited Input**.
2. Paste the client ID (and secret) under Device → Channels.
3. Tap **Sign in with Google**. Enter the code at google.com/device.

Gmail and Calendar then share that Google account. House → Behaviors **Ask before sending** stays on important actions: mail is a draft until you say send.

Without that client, Gmail still accepts an app password and Calendar still accepts a secret iCal address.

**Apple.** Sign in with Apple is not in this build. It needs an Apple Service ID we do not ship.

See [Claim this robot](claim) for the HomeKit-style setup code.
