// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package polling

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/pegnet/pegnet/common"
	log "github.com/sirupsen/logrus"
	"github.com/zpatrick/go-config"
)

const qlimit = 580 // Limit queries to once just shy of 10 minutes (600 seconds)

type PegAssets map[string]PegItem

func (p PegAssets) Clone(randomize float64) PegAssets {
	np := make(PegAssets)
	for _, asset := range common.AllAssets {
		np[asset] = p[asset].Clone(randomize)
	}

	return np
}

type PegItem struct {
	Value float64
	When  int64 // unix timestamp
}

func (p PegItem) Clone(randomize float64) PegItem {
	np := new(PegItem)
	np.Value = p.Value + p.Value*(randomize/2*rand.Float64()) - p.Value*(randomize/2*rand.Float64())
	np.Value = Round(np.Value)
	np.When = p.When
	return *np
}

var lastMutex sync.Mutex
var lastAnswer PegAssets //
var lastTime int64       // In seconds

var defaultDigitalAsset = "CoinCap"
var availableDigitalAssets = map[string]func(config *config.Config, peg PegAssets) error{
	"CoinCap": CoinCapInterface,
}

var defaultCurrencyAsset = "APILayer"
var availableCurrencyAssets = map[string]func(config *config.Config, peg PegAssets) error{
	"APILayer":          APILayerInterface,
	"ExchangeRatesAPI":  ExchangeRatesAPIInterface,
	"OpenExchangeRates": OpenExchangeRatesInterface,
}

var defaultMetalAsset = "Kitco"
var availableMetalAssets = map[string]func(config *config.Config, peg PegAssets) error{
	"Kitco": KitcoInterface,
}

func GetAssetsByWeight(config *config.Config, assets map[string]func(config *config.Config, peg PegAssets) error, default_asset string) []string {
	var result = []string{}
	for key := range assets {
		weight, _ := config.Int("Oracle." + key)
		for w := 0; w < weight; w++ {
			result = append(result, key)
		}
	}
	if len(result) == 0 {
		result = append(result, default_asset)
	}
	return result
}

func GetAvailableAssetsByWeight(config *config.Config) (string, string, string) {
	rand.Seed(time.Now().Unix())

	var digital_currencies = GetAssetsByWeight(config, availableDigitalAssets, defaultDigitalAsset)
	var currency_rates = GetAssetsByWeight(config, availableCurrencyAssets, defaultCurrencyAsset)
	var precious_metals = GetAssetsByWeight(config, availableMetalAssets, defaultMetalAsset)

	var digital_currencies_asset = digital_currencies[rand.Intn(len(digital_currencies))]
	var currency_rates_asset = currency_rates[rand.Intn(len(currency_rates))]
	var precious_metals_asset = precious_metals[rand.Intn(len(precious_metals))]

	// TODO: check if assets are in blacklist when running on production

	return digital_currencies_asset, currency_rates_asset, precious_metals_asset
}

func PullPEGAssets(config *config.Config) (pa PegAssets, err error) {
	// Prevent pounding of external APIs
	lastMutex.Lock()
	defer lastMutex.Unlock()
	now := time.Now().Unix()
	delta := now - lastTime

	// For testing, you can specify a randomization of the values returned by the oracles.
	// If the value specified isn't reasonable, then randomize is zero, and the values returned
	// are not changed.
	randomize, err := config.Float("Debug.Randomize")
	if err != nil && lastTime == 0 {
		log.WithError(err).Fatal(fmt.Sprintf("the config file doesn't have a valid Randomize value. %v", err))
	}

	if delta < qlimit && lastTime != 0 {
		pa := lastAnswer.Clone(randomize)
		return pa, nil
	}

	lastTime = now
	log.WithFields(log.Fields{
		"delta_time": delta,
	}).Info("Pulling PEG Asset data")

	peg := make(PegAssets)

	digital_currencies, currency_rates, precious_metals := GetAvailableAssetsByWeight(config)

	// digital currencies
	err = availableDigitalAssets[digital_currencies](config, peg)
	if err != nil {
		lastTime = 0 // Need to requery
		return
	}

	// currency rates
	err = availableCurrencyAssets[currency_rates](config, peg)
	if err != nil {
		lastTime = 0 // Need to requery
		return
	}

	// precious metals
	err = availableMetalAssets[precious_metals](config, peg)
	if err != nil {
		lastTime = 0 // Need to requery
		return
	}

	// debug
	fields := log.Fields{}
	for asset, v := range peg {
		fields[asset] = v.Value
	}
	log.WithFields(fields).Debug("Pulling PEG Asset data Result")

	lastAnswer = peg

	return peg.Clone(randomize), nil
}
