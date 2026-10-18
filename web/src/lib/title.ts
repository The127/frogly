// showTitle names the tab after the page, like "Garden · frogly"
export function showTitle(title?: string): void {
  document.title = title ? `${title} · frogly` : "frogly";
}
