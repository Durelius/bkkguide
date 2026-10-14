/** Pull coordinates out of a pasted Google Maps link, if it has any. */
export function coordsFromMapsUrl(url: string): { lat: number; lng: number } | null {
  // !3d…!4d… is the place itself, @… only the map's centre, so it goes first.
  const patterns = [/!3d(-?\d+\.\d+)!4d(-?\d+\.\d+)/, /@(-?\d+\.\d+),(-?\d+\.\d+)/, /[?&](?:q|query|ll)=(-?\d+\.\d+)(?:,|%2C)\s*(-?\d+\.\d+)/];
  for (const re of patterns) {
    const m = url.match(re);
    if (m) return { lat: parseFloat(m[1]), lng: parseFloat(m[2]) };
  }
  return null;
}

/** Share links (maps.app.goo.gl/…) only reveal coordinates once the server follows them. */
export function isShortMapsUrl(url: string): boolean {
  return /^https:\/\/(maps\.app\.goo\.gl|goo\.gl\/maps)\/\S+$/.test(url.trim());
}
