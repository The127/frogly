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

// visibilityText says who sees a project, as a sentence of its own
export function visibilityText(visibility: Visibility): string {
  const text = visibilityChoices.find((choice) => choice.value === visibility)?.text ?? visibility;

  return `${text.charAt(0).toUpperCase()}${text.slice(1)}.`;
}

// what a project's pages say when the project isn't there for the person
export const projectGone = { heading: "This project isn't here.", text: "It doesn't exist, or you aren't a member of it." };

// nameFrom reads the name the home page passed on. sv-router's route.search
// would turn a name like "2024" or "true" into a number or a boolean.
export function nameFrom(search: string): string {
  return new URLSearchParams(search).get("name") ?? "";
}

export function peopleBehind(people: number): string {
  return `${people} ${people === 1 ? "person" : "people"}`;
}

// one formatter for every idea of a garden, since making one isn't cheap
const days = new Map<string, Intl.DateTimeFormat>();

// The day an idea was planted, as the person reads dates. The locale and
// time zone are the browser's unless a test sets them.
export function plantedOn(plantedAt: string, locale?: string, timeZone?: string): string {
  const key = `${locale}|${timeZone}`;
  let day = days.get(key);
  if (!day) {
    day = new Intl.DateTimeFormat(locale, { day: "numeric", month: "short", year: "numeric", timeZone });
    days.set(key, day);
  }

  return day.format(new Date(plantedAt));
}
