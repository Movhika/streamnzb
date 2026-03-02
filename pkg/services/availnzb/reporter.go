package availnzb

import (
	"streamnzb/pkg/core/logger"
	"streamnzb/pkg/release"
	"streamnzb/pkg/session"
	"strings"
	"sync"
)

type ProviderHostsSource interface {
	GetProviderHosts() []string
}

type Reporter struct {
	client      *Client
	providerSrc ProviderHostsSource
	reported    sync.Map
}

func NewReporter(client *Client, providerSrc ProviderHostsSource) *Reporter {
	return &Reporter{client: client, providerSrc: providerSrc}
}

func (r *Reporter) ReportGood(sess *session.Session) {
	if _, loaded := r.reported.LoadOrStore(sess.ID, struct{}{}); loaded {
		return
	}
	logger.Info("Reporting good/streamable release to AvailNZB", "session", sess.ID)
	r.report(sess, true)
}

func (r *Reporter) ReportBad(sess *session.Session, reason string) {
	if reason != "" {
		logger.Info("Reporting bad/unstreamable release to AvailNZB", "session", sess.ID, "reason", reason)
	}
	r.report(sess, false)
}

func (r *Reporter) ReportRAR(sess *session.Session) {
	logger.Info("Reporting RAR release to AvailNZB (compression_type)", "session", sess.ID)
	r.report(sess, true)
}

func (r *Reporter) report(sess *session.Session, available bool) {
	if r.client == nil || r.client.BaseURL == "" {
		return
	}
	go func() {
		releaseURL := sess.ReleaseURL()
		if releaseURL == "" {
			return
		}
		if release.IsPrivateReleaseURL(releaseURL) {
			logger.Debug("Skipping AvailNZB report: release URL is private", "url", releaseURL)
			return
		}
		meta := ReportMeta{ReleaseName: sess.ReportReleaseName(), Size: sess.ReportSize()}
		if ids := sess.ContentIDs; ids != nil {

			if ids.TvdbID != "" && (ids.Season > 0 || ids.Episode > 0) {
				meta.TvdbID = ids.TvdbID
				meta.Season = ids.Season
				meta.Episode = ids.Episode
			} else if ids.ImdbID != "" {
				meta.ImdbID = ids.ImdbID
			} else if ids.TvdbID != "" {
				meta.TvdbID = ids.TvdbID
				meta.Season = ids.Season
				meta.Episode = ids.Episode
			}
		}
		if meta.ImdbID == "" && meta.TvdbID == "" {
			return
		}
		if meta.ReleaseName == "" {
			return
		}
		if sess.NZB != nil {
			meta.CompressionType = sess.NZB.CompressionType()
		}
		hosts := r.providerSrc.GetProviderHosts()
		if len(hosts) == 0 {
			return
		}
		if !available {
			_ = r.client.ReportAvailability(releaseURL, strings.Join(hosts, ","), false, meta)
		} else {
			_ = r.client.ReportAvailability(releaseURL, hosts[0], true, meta)
		}
	}()
}
