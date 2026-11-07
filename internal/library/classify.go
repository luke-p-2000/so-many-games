package library

import (
	"errors"
	"fmt"
)

// SaveClassification applies the staged statuses from the Classify screen (L7). Every
// unclassified, visible game must be included, so nothing is left half-classified.
func (s *Service) SaveClassification(staged map[int]Status) error {
	now := s.Now()
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`SELECT appid FROM game WHERE classified = 0 AND hidden = 0`)
	if err != nil {
		return err
	}
	var pending []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, id := range pending {
		st, ok := staged[id]
		if !ok {
			return errors.New("every unclassified game needs a status before Save all")
		}
		if !st.Valid() {
			return fmt.Errorf("invalid status for game %d", id)
		}
		if _, err := tx.Exec(`UPDATE game SET status = ?, classified = 1 WHERE appid = ?`, st, id); err != nil {
			return err
		}
		if err := changeStatus(tx, now, id, "", st, "classify"); err != nil {
			return err
		}
	}
	return tx.Commit()
}
