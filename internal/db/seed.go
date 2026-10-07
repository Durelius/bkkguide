package db

import (
	"database/sql"
	"fmt"

	"bkkguide/internal/auth"
)

// Development admin, created by -seed (never used in production, where the
// first admin comes from -create-admin).
const (
	DevAdminID       = "dev001"
	DevAdminName     = "Dev Admin"
	DevAdminPassword = "bkk-dev-password"
)

// Development seed data. Coordinates and hours are approximate placeholders;
// editors replace them with verified content through /admin.

type seedCategory struct {
	Slug, Name, Icon, Color string
}

type seedPlace struct {
	Category, Slug, Name, NameTH, Summary string
	Lat, Lng                              float64
	Station, Line                         string
	Walk, Price                           int
	DressCode, Tips, MustTry              string
	Featured                              bool
	Hours                                 [][3]string // weekday ("0"-"6" or "*"), opens, closes
}

var seedCategories = []seedCategory{
	{"shopping", "Shopping", "\uf290", "#BE3570"},
	{"restaurants", "Restaurants", "\uf0f5", "#C4532F"},
	{"culture", "Culture", "\uf19c", "#8A3F86"},
}

var seedPlaces = []seedPlace{
	// Restaurants
	{"restaurants", "jay-fai", "Jay Fai", "เจ๊ไฝ", "Michelin-starred street food. Famous for crab omelette cooked over charcoal.", 13.7524, 100.5045, "Sam Yot", "MRT Blue", 8, 4, "", "Queue early or book online; expect long waits.", "Crab omelette, drunken noodles", true, [][3]string{{"3", "09:00", "19:00"}, {"4", "09:00", "19:00"}, {"5", "09:00", "19:00"}, {"6", "09:00", "19:00"}}},
	{"restaurants", "thipsamai", "Thipsamai Pad Thai", "ทิพย์สมัย", "Old-school pad thai wrapped in a thin egg crepe.", 13.7527, 100.5049, "Sam Yot", "MRT Blue", 9, 2, "", "Fresh orange juice is a must.", "Pad thai in egg wrap", false, [][3]string{{"*", "09:00", "00:00"}}},
	{"restaurants", "krua-apsorn", "Krua Apsorn", "ครัวอัปสร", "Home-style royal Thai cooking loved by locals.", 13.7545, 100.5014, "Sam Yot", "MRT Blue", 15, 2, "", "", "Crab meat yellow curry, stir-fried crab with chilli", false, [][3]string{{"1", "10:30", "20:00"}, {"2", "10:30", "20:00"}, {"3", "10:30", "20:00"}, {"4", "10:30", "20:00"}, {"5", "10:30", "20:00"}, {"6", "10:30", "20:00"}}},
	{"restaurants", "somtam-nua", "Som Tam Nua", "ส้มตำนัว", "Spicy Isan papaya salad and fried chicken in Siam Square.", 13.7449, 100.5333, "Siam", "BTS Sukhumvit / Silom", 3, 1, "", "Ask for 'mai phet' if you can't handle the heat.", "Som tam, fried chicken", false, [][3]string{{"*", "10:45", "21:30"}}},
	{"restaurants", "wattana-panich", "Wattana Panich", "วัฒนาพานิช", "Beef noodle soup from a broth that has simmered for decades.", 13.7330, 100.5870, "Ekkamai", "BTS Sukhumvit", 12, 1, "", "", "Beef noodle soup", false, [][3]string{{"*", "09:00", "19:30"}}},

	// Shopping
	{"shopping", "chatuchak", "Chatuchak Weekend Market", "ตลาดนัดจตุจักร", "Over 10,000 stalls selling everything from vintage clothes to plants.", 13.7999, 100.5502, "Chatuchak Park", "MRT Blue", 3, 1, "", "Go early before the heat. Bring cash and water.", "", true, [][3]string{{"6", "09:00", "18:00"}, {"0", "09:00", "18:00"}}},
	{"shopping", "siam-paragon", "Siam Paragon", "สยามพารากอน", "Flagship mall with luxury brands, a huge food hall and an aquarium.", 13.7462, 100.5347, "Siam", "BTS Sukhumvit / Silom", 1, 4, "", "The basement food hall is great for a quick lunch.", "", false, [][3]string{{"*", "10:00", "22:00"}}},
	{"shopping", "mbk-center", "MBK Center", "เอ็ม บี เค เซ็นเตอร์", "Eight floors of phones, gadgets and bargains.", 13.7446, 100.5300, "National Stadium", "BTS Silom", 2, 1, "", "Haggling is expected at small stalls.", "", false, [][3]string{{"*", "10:00", "22:00"}}},
	{"shopping", "iconsiam", "ICONSIAM", "ไอคอนสยาม", "Riverside mega-mall with an indoor floating market.", 13.7267, 100.5103, "Charoen Nakhon", "Gold Line", 2, 3, "", "Take the free shuttle boat from Sathorn pier.", "", false, [][3]string{{"*", "10:00", "22:00"}}},
	{"shopping", "jodd-fairs", "Jodd Fairs", "จ๊อดแฟร์", "Night market with street food and second-hand stalls.", 13.7570, 100.5655, "Phra Ram 9", "MRT Blue", 5, 1, "", "", "", false, [][3]string{{"*", "16:00", "00:00"}}},

	// Culture
	{"culture", "wat-pho", "Wat Pho", "วัดโพธิ์", "Home of the 46-metre Reclining Buddha and Thai massage school.", 13.7465, 100.4927, "Sanam Chai", "MRT Blue", 5, 2, "Shoulders and knees covered. Shoes off inside halls.", "Don't point your feet at Buddha images.", "", true, [][3]string{{"*", "08:00", "18:30"}}},
	{"culture", "grand-palace", "The Grand Palace", "พระบรมมหาราชวัง", "Former royal residence and home of the Emerald Buddha.", 13.7500, 100.4913, "Sanam Chai", "MRT Blue", 12, 3, "Strict: long trousers or skirt, covered shoulders. No see-through clothing.", "Ignore strangers saying it's closed today. It's a common scam.", "", false, [][3]string{{"*", "08:30", "15:30"}}},
	{"culture", "wat-arun", "Wat Arun", "วัดอรุณ", "Temple of Dawn, best seen from the river at sunset.", 13.7437, 100.4889, "Sanam Chai", "MRT Blue", 10, 1, "Shoulders and knees covered.", "Cross the river by ferry from Tha Tien pier.", "", false, [][3]string{{"*", "08:00", "18:00"}}},
	{"culture", "jim-thompson-house", "Jim Thompson House", "บ้านจิมทอมป์สัน", "Teak houses and silk collection of the American who revived Thai silk.", 13.7491, 100.5283, "National Stadium", "BTS Silom", 5, 2, "", "Visits are by guided tour only.", "", false, [][3]string{{"*", "10:00", "18:00"}}},
	{"culture", "bacc", "Bangkok Art and Culture Centre", "หอศิลปวัฒนธรรมแห่งกรุงเทพมหานคร", "Free contemporary art exhibitions in a spiral gallery.", 13.7467, 100.5302, "National Stadium", "BTS Silom", 1, 1, "", "", "", false, [][3]string{{"2", "10:00", "20:00"}, {"3", "10:00", "20:00"}, {"4", "10:00", "20:00"}, {"5", "10:00", "20:00"}, {"6", "10:00", "20:00"}, {"0", "10:00", "20:00"}}},
}

// Seed inserts the development data if the database has no categories yet.
func Seed(conn *sql.DB) error {
	var n int
	if err := conn.QueryRow(`SELECT count(*) FROM categories`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	hash, err := auth.HashPassword(DevAdminPassword)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO admins (student_id, full_name, password_hash) VALUES (?, ?, ?)`, DevAdminID, DevAdminName, hash); err != nil {
		return err
	}

	catIDs := map[string]int64{}
	for i, c := range seedCategories {
		res, err := tx.Exec(`INSERT INTO categories (slug, name, icon, color, sort_order) VALUES (?, ?, ?, ?, ?)`, c.Slug, c.Name, c.Icon, c.Color, i)
		if err != nil {
			return err
		}
		catIDs[c.Slug], _ = res.LastInsertId()
	}

	for _, p := range seedPlaces {
		res, err := tx.Exec(`INSERT INTO places
			(slug, category_id, name, name_th, summary, lat, lng, nearest_station, station_line, walk_minutes, price_level, dress_code, etiquette_tips, must_try, status, featured)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'published', ?)`,
			p.Slug, catIDs[p.Category], p.Name, p.NameTH, p.Summary, p.Lat, p.Lng, p.Station, p.Line, p.Walk, p.Price, p.DressCode, p.Tips, p.MustTry, p.Featured)
		if err != nil {
			return fmt.Errorf("seed %s: %w", p.Slug, err)
		}
		id, _ := res.LastInsertId()
		for _, h := range p.Hours {
			days := []string{h[0]}
			if h[0] == "*" {
				days = []string{"0", "1", "2", "3", "4", "5", "6"}
			}
			for _, d := range days {
				if _, err := tx.Exec(`INSERT INTO place_hours (place_id, weekday, opens, closes) VALUES (?, ?, ?, ?)`, id, d, h[1], h[2]); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}
