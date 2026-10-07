import site from "../site.json";

// Change the name in web/site.json. The Go server reads the same file.
export const SITE_NAME: string = site.name;
export const SITE_TAGLINE: string = site.tagline;
export const BASE_URL: string = site.baseUrl;
