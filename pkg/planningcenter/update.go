package planningcenter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/paupin2/slides/pkg/config"
	"github.com/paupin2/slides/pkg/data"
	"github.com/rs/zerolog/log"
)

type vals map[string]interface{}

const (
	BaseURL  = "https://api.planningcenteronline.com"
	PageSize = 100
)

func Call(path string, params vals, reply interface{}) error {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, fmt.Sprint(v))
	}

	msg := log.With().Str("url", path+"?"+q.Encode()).Logger()
	cfg := config.Config.PlanningCenter
	if cfg.AppID == "" || cfg.Secret == "" {
		msg.Fatal().Msg("no user/password")
		return errors.New("bad user/password")
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u, err := url.Parse(BaseURL + path)
	if err != nil {
		msg.Err(err).Msg("bad path/url")
		return err
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		msg.Err(err).Msg("creating request")
		return err
	}

	req.SetBasicAuth(cfg.AppID, cfg.Secret)
	var resp *http.Response
	if resp, err = http.DefaultClient.Do(req); err != nil {
		msg.Err(err).Msg("doing request")
		return err
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(reply); err != nil {
		msg.Err(err).Msg("decoding request")
		return err
	}
	msg.Info().Msg("request")
	return nil
}

func Update() error {
	// see: https://developer.planning.center/docs/#/apps/services/2018-11-01/vertices/song
	log.Info().Msg("getting new songs")
	inserted, updated := 0, 0
	for offset := 0; ; offset += PageSize {
		var reply struct {
			Data  []Song
			Links struct {
				Self string `json:"self"`
				Next string `json:"next"`
			} `json:"links"`
			Meta struct {
				TotalCount int `json:"total_count"`
				Count      int `json:"count"`
				Next       struct {
					Offset int `json:"offset"`
				} `json:"next"`
			} `json:"meta"`
		}

		err := Call("/services/v2/songs", vals{
			"per_page": PageSize,
			// "order":    "-updated_at", // most recently-updated first
			"offset": offset,
		}, &reply)
		if err != nil {
			return err
		}

		for _, song := range reply.Data {
			if song.ID == "" {
				// empty id
				continue
			}

			existing := data.SongByExternalID(IDPrefix + song.ID)
			if existing != nil && existing.Modified.After(song.UpdatedAt()) {
				// local version is more recent
				continue
			}

			ds, err := song.Fetch()
			if err != nil {
				log.Err(err).Str("id", song.ID).Msg("error loading")
				return err
			}

			if existing != nil {
				ds.RowID = existing.RowID
			}

			// if err = ds.Check(); err != nil {
			// 	log.Warn().Err(err).Str("id", song.ID).Msg("skipping")
			// 	continue
			// }

			if !ds.Save() {
				return errors.New("error saving")
			}
			if existing != nil {
				updated++
			} else {
				inserted++
			}
		}

		if len(reply.Data) < PageSize {
			// not a full page: we've reached the end
			break
		}
	}
	log.Info().Int("inserted", inserted).Int("updated", updated).Msg("done")
	return nil

}
