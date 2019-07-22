// Copyright (c) of parts are held by the various contributors (see the CLA)
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package polling

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/pegnet/pegnet/common"

	"github.com/cenkalti/backoff"
	log "github.com/sirupsen/logrus"
	"github.com/zpatrick/go-config"
)

type CoinCapResponse struct {
	Data      []CoinCapRecord `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

type CoinCapRecord struct {
	ID                string `json:"id"`
	Rank              string `json:"rank"`
	Symbol            string `json:"symbol"`
	Name              string `json:"name"`
	Supply            string `json:"supply"`
	MaxSupply         string `json:"maxSupply"`
	MarketCapUSD      string `json:"marketCapUsd"`
	VolumeUSD24Hr     string `json:"volumeUsd24Hr"`
	PriceUSD          string `json:"priceUsd"`
	ChangePercent24Hr string `json:"changePercent24Hr"`
	VWAP24Hr          string `json:"vwap24Hr"`
}

func CallCoinCap(config *config.Config) (CoinCapResponse, error) {
	var CoinCapResponse CoinCapResponse

	operation := func() error {
		resp, err := http.Get("http://api.coincap.io/v2/assets?limit=500")
		if err != nil {
			log.WithError(err).Warning("Failed to get response from CoinCap")
			return err
		}
		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			return err
		} else if err = json.Unmarshal(body, &CoinCapResponse); err != nil {
			return err
		}
		return nil
	}

	err := backoff.Retry(operation, PollingExponentialBackOff())
	return CoinCapResponse, err
}

func HandleCoinCap(response CoinCapResponse, peg PegAssets) {

	var timestamp = response.Timestamp

	for _, currency := range response.Data {
		switch currency.Symbol {
		case "BTC", "XBT":
			value, err := strconv.ParseFloat(currency.PriceUSD, 64)
			peg["XBT"] = PegItem{Value: Round(value), When: timestamp}
			if err != nil {
				continue
			}
		case "BCH", "XBC":
			value, err := strconv.ParseFloat(currency.PriceUSD, 64)
			peg["XBC"] = PegItem{Value: Round(value), When: timestamp}
			if err != nil {
				continue
			}
		case "ZCASH", "ZEC":
			value, err := strconv.ParseFloat(currency.PriceUSD, 64)
			peg["ZCASH"] = PegItem{Value: Round(value), When: timestamp}
			if err != nil {
				continue
			}
		default:
			// See if the ticker is in our crypto currency list
			if common.AssetListContains(common.CryptoAssets, currency.Symbol) {
				value, err := strconv.ParseFloat(currency.PriceUSD, 64)
				peg[currency.Symbol] = PegItem{Value: Round(value), When: timestamp}
				if err != nil {
					continue
				}
			}
		}
	}

}

func CoinCapInterface(config *config.Config, peg PegAssets) {
	log.Debug("Pulling Asset data from CoinCap")
	CoinCapResponse, err := CallCoinCap(config)
	if err != nil {
		log.WithError(err).Fatal("Failed to access CoinCap")
	} else {
		HandleCoinCap(CoinCapResponse, peg)
	}
}
