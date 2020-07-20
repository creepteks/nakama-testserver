// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	timestampOp_tick int64 = 0
)

var left int = 0

func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {

	logger.Info("neon backend loaded.")

	// if err := initializer.RegisterRpc("go_echo_sample", rpcEcho); err != nil {
	// 	return err
	// }
	// if err := initializer.RegisterBeforeRt("ChannelJoin", beforeChannelJoin); err != nil {
	// 	return err
	// }
	// if err := initializer.RegisterAfterGetAccount(afterGetAccount); err != nil {
	// 	return err
	// }
	if err := initializer.RegisterMatch("davaaPvP", CreateMatchInternal); err != nil {
		return err
	}

	if err := initializer.RegisterMatchmakerMatched(DoMatchmaking); err != nil {
		logger.Error("Unable to register: %v", err)
		return err
	}

	// if err := initializer.RegisterEventSessionStart(eventSessionStart); err != nil {
	// 	return err
	// }
	// if err := initializer.RegisterEventSessionEnd(eventSessionEnd); err != nil {
	// 	return err
	// }
	// if err := initializer.RegisterEvent(func(ctx context.Context, logger runtime.Logger, evt *api.Event) {
	// 	logger.Info("Received event: %+v", evt)
	// }); err != nil {
	// 	return err
	// }
	return nil
}

func DoMatchmaking(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, entries []runtime.MatchmakerEntry) (string, error) {
	for _, e := range entries {
		logger.Info("Matched user '%s' named '%s'", e.GetPresence().GetUserId(), e.GetPresence().GetUsername())
		for k, v := range e.GetProperties() {
			logger.Info("Matched on '%s' value '%v'", k, v)
		}
	}

	matchId, err := nk.MatchCreate(ctx, "davaaPvP", map[string]interface{}{"invited": entries, "debug": true})
	if err != nil {
		return "", err
	}

	return matchId, nil
}

func CreateMatchInternal(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
	return &Match{}, nil
}

// func rpcEcho(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
// 	logger.Print("RUNNING IN GO")
// 	return payload, nil
// }

// func beforeChannelJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, envelope *rtapi.Envelope) (*rtapi.Envelope, error) {
// 	logger.Printf("Intercepted request to join channel '%v'", envelope.GetChannelJoin().Target)
// 	return envelope, nil
// }

// func afterGetAccount(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, in *api.Account) error {
// 	logger.Printf("Intercepted response to get account '%v'", in)
// 	return nil
// }

type MatchState struct {
	debug     bool
	presences map[string]runtime.Presence
	opponents map[string]runtime.Presence
}

type Match struct{}

func (m *Match) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {

	left = 0

	var debug bool
	if d, ok := params["debug"]; ok {
		if dv, ok := d.(bool); ok {
			debug = dv
		}
	}
	state := &MatchState{
		debug:     debug,
		presences: make(map[string]runtime.Presence),
		opponents: make(map[string]runtime.Presence),
	}

	if state.debug {
		logger.Printf("match init, starting with debug: %v", state.debug)
	}

	entries := params["invited"].([]runtime.MatchmakerEntry)
	for i := 0; i < len(entries); i++ {
		p := entries[(i+1)%len(entries)].GetPresence()
		state.opponents[entries[i].GetPresence().GetUsername()] = p
		logger.Info("setting %v 's opponent : %v", entries[i].GetPresence().GetUsername(), entries[(i+1)%len(entries)].GetPresence().GetUsername())
	}

	label := "skill=100-150"
	tickRate := 20 // ticks per second, so interval of two ticks is  50ms
	return state, tickRate, label
}

func (m *Match) MatchJoinAttempt(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presence runtime.Presence, metadata map[string]string) (interface{}, bool, string) {
	if state.(*MatchState).debug {
		logger.Printf("match join attempt username %v user_id %v session_id %v node %v with metadata %v", presence.GetUsername(), presence.GetUserId(), presence.GetSessionId(), presence.GetNodeId(), metadata)
	}

	return state, true, ""
}

func (m *Match) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	mState := state.(*MatchState)
	if mState.debug {
		for _, presence := range presences {
			logger.Printf("match join username %v user_id %v session_id %v node %v", presence.GetUsername(), presence.GetUserId(), presence.GetSessionId(), presence.GetNodeId())
		}
	}

	return mState
}

func (m *Match) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	if state.(*MatchState).debug {
		for _, presence := range presences {
			logger.Printf("match leave username %v user_id %v session_id %v node %v", presence.GetUsername(), presence.GetUserId(), presence.GetSessionId(), presence.GetNodeId())
			left++
			logger.Info("number of leaves %v", left)
		}
	}

	if left > 0 {
		logger.Info("someone left, returning null")
		return nil
	}

	return state
}

func (m *Match) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
	// if left > 0 {
	// 	logger.Info("match Terminated")
	// 	return nil
	// }

	mState, _ := state.(*MatchState)

	// if state.(*MatchState).debug {
	// 	logger.Printf("match loop match_id %v tick %v", ctx.Value(runtime.RUNTIME_CTX_MATCH_ID), tick)
	// 	logger.Printf("match loop match_id %v message count %v", ctx.Value(runtime.RUNTIME_CTX_MATCH_ID), len(messages))
	// }

	var bytes, _ = json.Marshal(tick)
	if tick%20 == 0 {
		dispatcher.BroadcastMessage(timestampOp_tick, bytes, nil, nil, true)
	}

	for _, message := range messages {

		target := mState.opponents[message.GetUsername()]
		if state.(*MatchState).debug {
			logger.Info("Received %v from %v to ", string(message.GetData()), message.GetUsername(), target.GetUsername())
		}
		dispatcher.BroadcastMessage(message.GetOpCode(), message.GetData(), []runtime.Presence{target}, nil, false)
	}
	return mState
}

func (m *Match) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
	if state.(*MatchState).debug {
		logger.Printf("match terminate match_id %v tick %v", ctx.Value(runtime.RUNTIME_CTX_MATCH_ID), tick)
		logger.Printf("match terminate match_id %v grace seconds %v", ctx.Value(runtime.RUNTIME_CTX_MATCH_ID), graceSeconds)
	}

	return nil
}

// func eventSessionStart(ctx context.Context, logger runtime.Logger, evt *api.Event) {
// 	logger.Printf("session start %v %v", ctx, evt)
// }

// func eventSessionEnd(ctx context.Context, logger runtime.Logger, evt *api.Event) {
// 	logger.Printf("session end %v %v", ctx, evt)
// }
