package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/sirupsen/logrus"
)

type KeeperSyncer struct {
	KeeperEndpoint string
	SyncInterval   time.Duration
	Store          *Store
	Logger         *logrus.Logger

	httpClient *http.Client
}

func (s *KeeperSyncer) Start(ctx context.Context) {
	s.httpClient = http.DefaultClient

	ticker := time.NewTicker(s.SyncInterval)

	go func() {
		if err := s.Sync(); err != nil {
			s.Logger.WithError(err).WithField("keeperEndpoint", s.KeeperEndpoint).
				Error("failed to do the initial sync with Keeper")
		}

		for {
			select {
			case <-ticker.C:
				s.Logger.WithField("keeperEndpoint", s.KeeperEndpoint).Trace("Syncing Keeper merge status/history...")
				if err := s.Sync(); err != nil {
					s.Logger.WithError(err).WithField("keeperEndpoint", s.KeeperEndpoint).
						Error("failed to sync Keeper merge status/history")
				}
			case <-ctx.Done():
				s.Logger.Info("KeeperSyncer exiting...")
				return
			}
		}
	}()
}

func (s *KeeperSyncer) Sync() error {
	const (
		keeperPoolsPath   = "/"
		keeperHistoryPath = "/history"
	)

	{
		resp, err := s.get(keeperPoolsPath)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		dec.UseNumber()
		lhPools, err := gabs.ParseJSONDecoder(dec)
		if err != nil {
			return err
		}

		pools := MergePoolsFromLighthousePools(lhPools)
		s.Store.SetMergeStatus(pools)
	}

	{
		resp, err := s.get(keeperHistoryPath)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(resp.Body)
		dec.UseNumber()
		lhRecords, err := gabs.ParseJSONDecoder(dec)
		if err != nil {
			return err
		}

		records := MergeRecordsFromLighthouseRecords(lhRecords)
		s.Store.SetMergeHistory(records)
	}

	return nil
}

func (s *KeeperSyncer) get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, s.KeeperEndpoint+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "lighthouse-webui-plugin")

	return s.httpClient.Do(req)
}
