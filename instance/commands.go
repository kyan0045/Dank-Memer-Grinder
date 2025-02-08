package instance

import (
	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/BridgeSenseDev/Dank-Memer-Grinder/utils"
)

func (in *Instance) CommandsLoop() {
	ticker := time.NewTicker(2500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			breakOff := in.Cfg.Cooldowns.BreakTime.MinHours == 0 && in.Cfg.Cooldowns.BreakTime.MaxHours == 0

			if in.State == "healthy" {
				in.State = "running"
				if !breakOff {
					duration := time.Duration((utils.Rng.Float32()*(in.Cfg.Cooldowns.BreakCooldown.MaxHours-in.Cfg.Cooldowns.BreakCooldown.MinHours) + in.Cfg.Cooldowns.BreakCooldown.MinHours) * float32(time.Hour))
					in.BreakUpdateTime = time.Now().Add(duration)
				}
				application.Get().EmitEvent("breakUpdate", in.AccountCfg.Token, in.State, in.BreakUpdateTime)
			} else if in.State == "running" && !breakOff {
				if time.Now().After(in.BreakUpdateTime) {
					in.State = "sleeping"
					duration := time.Duration((utils.Rng.Float32()*(in.Cfg.Cooldowns.BreakTime.MaxHours-in.Cfg.Cooldowns.BreakTime.MinHours) + in.Cfg.Cooldowns.BreakTime.MinHours) * float32(time.Hour))
					in.BreakUpdateTime = time.Now().Add(duration)
					in.PauseCommands(true)
					application.Get().EmitEvent("breakUpdate", in.AccountCfg.Token, in.State, in.BreakUpdateTime)
				}
			} else if in.State == "sleeping" && !breakOff {
				if time.Now().After(in.BreakUpdateTime) {
					in.State = "running"
					duration := time.Duration((utils.Rng.Float32()*(in.Cfg.Cooldowns.BreakCooldown.MaxHours-in.Cfg.Cooldowns.BreakCooldown.MinHours) + float32(in.Cfg.Cooldowns.BreakCooldown.MinHours)) * float32(time.Hour))
					in.BreakUpdateTime = time.Now().Add(duration)
					in.UnpauseCommands()
					application.Get().EmitEvent("breakUpdate", in.AccountCfg.Token, in.State, in.BreakUpdateTime)
				}
			}

			if !in.IsPaused() && in.Cfg.State && in.AccountCfg.State {
				commandsMap := in.Cfg.Commands.GetCommandsMap()
				for command := range commandsMap {
					select {
					case <-in.StopChan:
						return
					default:
						minDelay := in.Cfg.Cooldowns.CommandInterval.MinDelay
						maxDelay := in.Cfg.Cooldowns.CommandInterval.MaxDelay
						<-utils.Sleep(time.Duration(rand.Intn(maxDelay-minDelay)+minDelay) * time.Millisecond)

						if !shouldExecuteCommand(in, command) {
							continue
						}

						in.LastRan[command] = time.Now().Add(2 * time.Second)
						var err error
						if command == "Blackjack" {
							err = in.SendCommand("blackjack", map[string]string{"bet": in.Cfg.Commands.Blackjack.Amount}, false)
						} else if command == "Pets" {
							err = in.SendSubCommand("pets", "care", nil, false)
						} else if command == "Work" {
							err = in.SendSubCommand("work", "shift", nil, false)
						} else if command == "Deposit" {
							err = in.SendCommand("deposit", map[string]string{"amount": "max"}, false)
						} else if command == "Fish" {
							err = in.SendSubCommand("fish", "catch", nil, false)
						} else {
							err = in.SendCommand(strings.ToLower(command), nil, false)
						}

						if err != nil {
							utils.Log(utils.Others, utils.Error, in.SafeGetUsername(), fmt.Sprintf("Failed to send %s command: %s", command, err.Error()))
						}
					}
				}
			} else {
				<-utils.Sleep(1 * time.Second)
			}
		case <-in.StopChan:
			return
		}
	}
}

func shouldExecuteCommand(in *Instance, command string) bool {
	commandConfig := in.Cfg.Commands.GetCommandsMap()[command]
	if time.Since(in.LastRan[command]) < time.Duration(commandConfig.Delay)*time.Second || !commandConfig.State || in.IsPaused() || !in.Cfg.State || !in.AccountCfg.State {
		return false
	}

	switch command {
	case "Blackjack":
		return !in.Cfg.Commands.Blackjack.ManuallyRunCommands
	case "Profile":
		val := reflect.ValueOf(in.Cfg.AutoUse)
		for i := 0; i < val.NumField(); i++ {
			if val.Field(i).FieldByName("State").Bool() {
				return true
			}
		}
		return false
	default:
		return true
	}
}
