export const HOUSEHOLD_ROLES = [
  { id: "owner", title: "Owner", hint: "This is their robot. Can install plugins and sign in accounts." },
  { id: "family", title: "Family", hint: "Talk, mail, calendar. Cannot claim or install plugins." },
  { id: "kid", title: "Kid", hint: "No mail, calendar, or computer-use — same bound as Kids around." },
  { id: "guest", title: "Guest", hint: "Greet and chat. No connectors." },
] as const;

export type HouseholdRole = (typeof HOUSEHOLD_ROLES)[number]["id"];

export function roleTitle(role?: string): string {
  const hit = HOUSEHOLD_ROLES.find((r) => r.id === role);
  return hit ? hit.title : "Friend";
}
