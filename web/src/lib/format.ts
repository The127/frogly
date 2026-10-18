import type { Joining, Visibility } from "./api";

// the choices a page offers for who sees a project and who can join it,
// the same wherever they're chosen
export const visibilityChoices: { value: Visibility; text: string }[] = [
  { value: "private", text: "private, only its members see it" },
  { value: "public", text: "public, everyone in frogly sees its name" },
];

export const joiningChoices: { value: Joining; text: string }[] = [
  { value: "open", text: "anyone, right away" },
  { value: "request", text: "people who ask, once a member says yes" },
  { value: "closed", text: "only people a member adds" },
];

// nameFrom reads the name the home page passed on. sv-router's route.search
// would turn a name like "2024" or "true" into a number or a boolean.
export function nameFrom(search: string): string {
  return new URLSearchParams(search).get("name") ?? "";
}

export function peopleBehind(people: number): string {
  return `${people} ${people === 1 ? "person" : "people"}`;
}

// The day an idea was planted, as the person reads dates. The locale and
// time zone are the browser's unless a test sets them.
export function plantedOn(plantedAt: string, locale?: string, timeZone?: string): string {
  return new Date(plantedAt).toLocaleDateString(locale, { day: "numeric", month: "short", year: "numeric", timeZone });
}
