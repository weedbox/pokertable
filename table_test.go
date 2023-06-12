package pokertable

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_InvalidGamePlayerIndex(t *testing.T) {
	jsonStr := `
	{"id":"bbb14f64-862f-4d39-b444-ca6a5aa53504","meta":{"short_id":"ABC123","code":"01","name":"table name","invitation_code":"come_to_play","competition_meta":{"id":"competition id","rule":"default","mode":"mtt","max_duration_mins":60,"table_max_seat_count":9,"table_min_playing_count":2,"min_chips_unit":10,"blind":{"id":"9963ee6a-6ef5-43bd-9167-db6c6cfabb1f","name":"blind name","initial_level":1,"final_buy_in_level":2,"dealer_blind_times":0,"levels":[{"level":1,"sb_chips":10,"bb_chips":20,"ante_chips":0,"duration_mins":10},{"level":2,"sb_chips":20,"bb_chips":30,"ante_chips":0,"duration_mins":10},{"level":3,"sb_chips":30,"bb_chips":40,"ante_chips":0,"duration_mins":10}]},"action_time_secs":0}},"state":{"game_count":2,"start_game_at":1686421171,"blind_state":{"final_buy_in_level_idx":1,"initial_level":1,"current_level_index":0,"level_states":[{"level":1,"sb_chips":10,"bb_chips":20,"ante_chips":0,"duration_mins":10,"level_end_at":1686421771},{"level":2,"sb_chips":20,"bb_chips":30,"ante_chips":0,"duration_mins":10,"level_end_at":1686422371},{"level":3,"sb_chips":30,"bb_chips":40,"ante_chips":0,"duration_mins":10,"level_end_at":1686422971}]},"current_dealer_seat_index":1,"current_bb_seat_index":4,"player_seat_map":[1,0,-1,-1,2,-1,-1,-1,-1],"player_states":[{"player_id":"Jeffrey","seat_index":1,"positions":["dealer","sb"],"is_participated":true,"is_between_dealer_bb":false,"bankroll":6000},{"player_id":"Chuck","seat_index":0,"positions":[],"is_participated":false,"is_between_dealer_bb":false,"bankroll":0},{"player_id":"Fred","seat_index":4,"positions":["bb"],"is_participated":true,"is_between_dealer_bb":false,"bankroll":3000}],"game_player_indexes":[0,2],"status":"TableGame_MatchOpen"},"update_at":1686421171}
	`

	var table Table
	err := json.Unmarshal([]byte(jsonStr), &table)
	assert.Nil(t, err)

	assert.Equal(t, 0, table.GamePlayerIndex("Jeffrey"))
	assert.Equal(t, 1, table.GamePlayerIndex("Fred"))
	assert.Equal(t, -1, table.GamePlayerIndex("Chuck"))
}
