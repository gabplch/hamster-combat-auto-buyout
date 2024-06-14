package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	RequestContentType = "application/json"
	ApiHost            = "https://api.hamsterkombat.io"
	UserTokenEnvName   = "token"
	UpgradesPath       = "/clicker/upgrades-for-buy"
	UserProfilePath    = "/clicker/sync"
	BuyUpgradePath     = "/clicker/buy-upgrade"
)

type UpgradeForBuy struct {
	Id                   string      `json:"id"`
	Name                 string      `json:"name"`
	Price                int         `json:"price"`
	ProfitPerHour        int         `json:"profitPerHour"`
	Condition            interface{} `json:"condition"`
	Section              string      `json:"section"`
	Level                int         `json:"level"`
	CurrentProfitPerHour int         `json:"currentProfitPerHour"`
	ProfitPerHourDelta   int         `json:"profitPerHourDelta"`
	IsAvailable          bool        `json:"isAvailable"`
	IsExpired            bool        `json:"isExpired"`
	MaxLevel             int         `json:"maxLevel"`
	CooldownSeconds      int         `json:"cooldownSeconds"`
}
type UserProfile struct {
	Id             string  `json:"id"`
	TotalCoins     float64 `json:"totalCoins"`
	BalanceCoins   float64 `json:"balanceCoins"`
	Level          int     `json:"level"`
	AvailableTaps  int     `json:"availableTaps"`
	LastSyncUpdate int     `json:"lastSyncUpdate"`
}

func CalculateProfit(upgrade UpgradeForBuy) float64 {
	return float64(upgrade.ProfitPerHour) / float64(upgrade.Price)
}

type UpgradesForBuyResponse struct {
	UpgradesForBuy []UpgradeForBuy
}
type UserProfileResponse struct {
	ClickerUser UserProfile
}
type BuyUpgradeResponse struct {
	ClickerUser    UserProfile
	UpgradesForBuy []UpgradeForBuy
}

func main() {
	os.Setenv(UserTokenEnvName, os.Args[2])

	userProfile := FetchUserProfile()

	upgrades := FetchUpgrades()
	upgrades = FilterUpgrades(upgrades, userProfile.BalanceCoins)

	for len(upgrades) > 0 {
		upgrade := GetMostProfitableUpgrade(upgrades)
		upgrades, userProfile = BuyUpgrade(upgrade)

		upgrades = FilterUpgrades(upgrades, userProfile.BalanceCoins)

		fmt.Println("Bought upgrade " + upgrade.Name)
	}

	os.Clearenv()
}

func FetchUpgrades() []UpgradeForBuy {
	client := &http.Client{}

	req, _ := http.NewRequest("POST", ApiHost+UpgradesPath, nil)
	req.Header.Add("Authorization", "Bearer "+os.Getenv(UserTokenEnvName))

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	var result UpgradesForBuyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Can not unmarshal JSON")
	}

	return result.UpgradesForBuy
}

func FilterUpgrades(upgradesForBuy []UpgradeForBuy, balance float64) []UpgradeForBuy {
	var availableUpgrades []UpgradeForBuy

	for _, upgradeForBuy := range upgradesForBuy {
		// skip non available upgrades
		if !(upgradeForBuy.IsAvailable && !upgradeForBuy.IsExpired) {
			continue
		}

		// skip upgrades, that are already on max level
		if upgradeForBuy.MaxLevel > 0 && upgradeForBuy.MaxLevel <= upgradeForBuy.Level {
			continue
		}

		// skip expensive upgrades
		if balance < float64(upgradeForBuy.Price) {
			continue
		}

		// skip upgrades with cooldown
		if upgradeForBuy.CooldownSeconds > 0 {
			continue
		}

		availableUpgrades = append(availableUpgrades, upgradeForBuy)
	}

	return availableUpgrades
}

func GetMostProfitableUpgrade(upgradesForBuy []UpgradeForBuy) UpgradeForBuy {
	var mostProfitableUpgrade UpgradeForBuy = upgradesForBuy[0]

	for _, upgrade := range upgradesForBuy {
		if CalculateProfit(upgrade) > CalculateProfit(mostProfitableUpgrade) {
			mostProfitableUpgrade = upgrade
		}
	}

	return mostProfitableUpgrade
}

func FetchUserProfile() UserProfile {
	client := &http.Client{}

	req, _ := http.NewRequest("POST", ApiHost+UserProfilePath, nil)
	req.Header.Add("Authorization", "Bearer "+os.Getenv(UserTokenEnvName))

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	var result UserProfileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Can not unmarshal JSON")
	}

	return result.ClickerUser
}

func BuyUpgrade(upgrade UpgradeForBuy) ([]UpgradeForBuy, UserProfile) {
	location, err := time.LoadLocation("Europe/Kyiv")
	currentTime := time.Now().In(location)

	client := &http.Client{}

	requestBody := []byte(fmt.Sprintf(`{"timestamp": %d,"upgradeId": "%s"}`, currentTime.Unix(), upgrade.Id))

	req, _ := http.NewRequest("POST", ApiHost+BuyUpgradePath, bytes.NewBuffer(requestBody))
	req.Header.Add("Authorization", "Bearer "+os.Getenv(UserTokenEnvName))
	req.Header.Add("Content-Type", RequestContentType)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	var result BuyUpgradeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Can not unmarshal JSON")
	}

	return result.UpgradesForBuy, result.ClickerUser
}
