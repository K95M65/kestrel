import indexMd from "@wiki/index.md?raw";
import gettingStartedMd from "@wiki/getting-started.md?raw";
import talkMd from "@wiki/talk.md?raw";
import voiceMd from "@wiki/voice.md?raw";
import peopleMd from "@wiki/people.md?raw";
import usesMd from "@wiki/uses.md?raw";
import houseMd from "@wiki/house.md?raw";
import buddyMd from "@wiki/buddy.md?raw";
import telegramMd from "@wiki/telegram.md?raw";
import dummyHostMd from "@wiki/dummy-host.md?raw";
import safetyMd from "@wiki/safety.md?raw";

export type WikiPage = {
  slug: string;
  title: string;
  body: string;
};

export const WIKI_PAGES: WikiPage[] = [
  { slug: "index", title: "Start here", body: indexMd },
  { slug: "getting-started", title: "Get started", body: gettingStartedMd },
  { slug: "talk", title: "Talk", body: talkMd },
  { slug: "voice", title: "Voice", body: voiceMd },
  { slug: "people", title: "People", body: peopleMd },
  { slug: "uses", title: "Uses", body: usesMd },
  { slug: "house", title: "House", body: houseMd },
  { slug: "buddy", title: "Kestrel Buddy", body: buddyMd },
  { slug: "telegram", title: "Telegram", body: telegramMd },
  { slug: "dummy-host", title: "Dummy host", body: dummyHostMd },
  { slug: "safety", title: "Kids and safety", body: safetyMd },
];

export function wikiPage(slug: string | null | undefined): WikiPage {
  const id = (slug || "index").trim();
  return WIKI_PAGES.find((p) => p.slug === id) ?? WIKI_PAGES[0];
}

export function isWikiSlug(href: string): boolean {
  return WIKI_PAGES.some((p) => p.slug === href);
}
