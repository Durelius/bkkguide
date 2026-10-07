import { Link } from "react-router-dom";
import { SITE_NAME, SITE_TAGLINE } from "../config";
import { SunriseMark } from "./Logo";
import "./Footer.css";

export function Footer() {
  return (
    <footer className="footer">
      <div className="footer__brand">
        <SunriseMark size={36} />
        <div>
          <p className="footer__name">{SITE_NAME}</p>
          <p className="footer__tagline">{SITE_TAGLINE}</p>
        </div>
      </div>
      <nav className="footer__links" aria-label="Footer">
        <Link to="/admin/login">Admin login</Link>
      </nav>
    </footer>
  );
}
