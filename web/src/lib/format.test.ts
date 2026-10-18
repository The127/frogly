import { describe, expect, it } from "vitest";

import { nameFrom, peopleBehind, plantedOn } from "./format";

describe("nameFrom", () => {
  it.each(["Summer fair", "2024", "007", "true", " ", "a&b=c"])("keeps %j as the text it is", (name) => {
    // act
    const read = nameFrom("?" + new URLSearchParams({ name }).toString());

    // assert
    expect(read).toBe(name);
  });

  it("is empty without a name", () => {
    // act
    const read = nameFrom("");

    // assert
    expect(read).toBe("");
  });
});

describe("peopleBehind", () => {
  it.each([
    { people: 1, text: "1 person" },
    { people: 2, text: "2 people" },
  ])("says $text", ({ people, text }) => {
    // act
    const said = peopleBehind(people);

    // assert
    expect(said).toBe(text);
  });
});

describe("plantedOn", () => {
  it("gives the day in the person's own time zone", () => {
    // act
    const day = plantedOn("2026-10-15T23:30:00Z", "en-GB", "Europe/Berlin");

    // assert
    expect(day).toBe("16 Oct 2026");
  });
});
