package it

import (
	"context"
	"testing"
	"time"

	"github.com/atlassian/smith"
	"github.com/atlassian/smith/examples/tprattribute"

	"github.com/ash2k/stager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTprAttribute(t *testing.T) {
	sleeper := &tprattribute.Sleeper{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       tprattribute.SleeperResourceKind,
			APIVersion: tprattribute.SleeperResourceGroupVersion,
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "sleeper1",
		},
		Spec: tprattribute.SleeperSpec{
			SleepFor:      1, // seconds,
			WakeupMessage: "Hello, Infravators!",
		},
	}
	bundle := &smith.Bundle{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       smith.BundleResourceKind,
			APIVersion: smith.BundleResourceGroupVersion,
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "bundle-attribute",
		},
		Spec: smith.BundleSpec{
			Resources: []smith.Resource{
				{
					Name: smith.ResourceName(sleeper.Name),
					Spec: sleeper,
				},
			},
		},
	}
	SetupApp(t, bundle, false, true, testTprAttribute, sleeper)
}

func testTprAttribute(t *testing.T, ctxTest context.Context, cfg *Config, args ...interface{}) {
	sleeper := args[0].(*tprattribute.Sleeper)
	sClient, err := tprattribute.GetSleeperTprClient(cfg.Config, sleeperScheme())
	require.NoError(t, err)

	stgr := stager.New()
	defer stgr.Shutdown()
	stage := stgr.NextStage()
	stage.StartWithContext(func(ctx context.Context) {
		apl := tprattribute.App{
			RestConfig: cfg.Config,
		}
		if e := apl.Run(ctx); e != context.Canceled && e != context.DeadlineExceeded {
			assert.NoError(t, e)
		}
	})

	ctxTimeout, cancel := context.WithTimeout(ctxTest, time.Duration(sleeper.Spec.SleepFor+3)*time.Second)
	defer cancel()

	AssertBundle(t, ctxTimeout, cfg.Store, cfg.Namespace, cfg.Bundle, "")

	var sleeperObj tprattribute.Sleeper
	require.NoError(t, sClient.Get().
		Context(ctxTest).
		Namespace(cfg.Namespace).
		Resource(tprattribute.SleeperResourcePath).
		Name(sleeper.Name).
		Do().
		Into(&sleeperObj))

	assert.Equal(t, map[string]string{
		smith.BundleNameLabel: cfg.Bundle.Name,
	}, sleeperObj.Labels)
	assert.Equal(t, tprattribute.Awake, sleeperObj.Status.State)
}
