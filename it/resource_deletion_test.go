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
	api_v1 "k8s.io/client-go/pkg/api/v1"
)

func TestResourceDeletion(t *testing.T) {
	cm := &api_v1.ConfigMap{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "cm",
		},
		Data: map[string]string{
			"a": "b",
		},
	}
	sleeper := &tprattribute.Sleeper{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       tprattribute.SleeperResourceKind,
			APIVersion: tprattribute.SleeperResourceGroupVersion,
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "sleeper2",
		},
		Spec: tprattribute.SleeperSpec{
			SleepFor:      1, // seconds,
			WakeupMessage: "Hello there!",
		},
	}
	bundle := &smith.Bundle{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       smith.BundleResourceKind,
			APIVersion: smith.BundleResourceGroupVersion,
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "bundle",
		},
		Spec: smith.BundleSpec{
			Resources: []smith.Resource{
				{
					Name: smith.ResourceName(cm.Name),
					Spec: cm,
				},
				{
					Name: smith.ResourceName(sleeper.Name),
					Spec: sleeper,
				},
			},
		},
	}
	SetupApp(t, bundle, false, false, testResourceDeletion, cm, sleeper)
}

func testResourceDeletion(t *testing.T, ctxTest context.Context, cfg *Config, args ...interface{}) {
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

	cm := args[0].(*api_v1.ConfigMap)
	sleeper := args[1].(*tprattribute.Sleeper)

	cmClient := cfg.Clientset.CoreV1().ConfigMaps(cfg.Namespace)
	sClient, err := tprattribute.GetSleeperTprClient(cfg.Config, sleeperScheme())
	require.NoError(t, err)

	// Create orphaned ConfigMap
	cmActual, err := cmClient.Create(cm)
	require.NoError(t, err)
	cfg.cleanupLater(cmActual)

	// Create orphaned Sleeper
	sleeperActual := &tprattribute.Sleeper{}
	err = sClient.Post().
		Context(ctxTest).
		Namespace(cfg.Namespace).
		Resource(tprattribute.SleeperResourcePath).
		Body(sleeper).
		Do().
		Into(sleeperActual)
	require.NoError(t, err)
	cfg.cleanupLater(sleeperActual)

	// Create Bundle with same resources
	bundleActual := &smith.Bundle{}
	cfg.createObject(ctxTest, cfg.Bundle, bundleActual, smith.BundleResourcePath, cfg.BundleClient)
	cfg.CreatedBundle = bundleActual

	time.Sleep(1 * time.Second) // TODO this should be removed once race with tpr informer is fixed "no informer for tpr.atlassian.com/v1, Kind=Sleeper is registered"

	// Bundle should be in Error=true state
	obj, err := cfg.Store.AwaitObjectCondition(ctxTest, smith.BundleGVK, cfg.Namespace, cfg.Bundle.Name, isBundleError)
	require.NoError(t, err)
	bundleActual = obj.(*smith.Bundle)

	assertCondition(t, bundleActual, smith.BundleReady, smith.ConditionFalse)
	assertCondition(t, bundleActual, smith.BundleInProgress, smith.ConditionFalse)
	cond := assertCondition(t, bundleActual, smith.BundleError, smith.ConditionTrue)
	if cond != nil {
		assert.Equal(t, "TerminalError", cond.Reason)
		assert.Equal(t, "object /v1, Kind=ConfigMap \"cm\" is not owned by the Bundle", cond.Message)
	}

	// Delete conflicting ConfigMap
	trueVar := true
	cmActual.OwnerReferences = []meta_v1.OwnerReference{
		{
			APIVersion: smith.BundleResourceGroupVersion,
			Kind:       smith.BundleResourceKind,
			Name:       bundleActual.Name,
			UID:        bundleActual.UID,
			Controller: &trueVar,
		},
	}
	err = cmClient.Delete(cmActual.Name, &meta_v1.DeleteOptions{
		Preconditions: &meta_v1.Preconditions{
			UID: &cmActual.UID,
		},
	})
	require.NoError(t, err)

	err = sClient.Delete().
		Context(ctxTest).
		Namespace(cfg.Namespace).
		Resource(tprattribute.SleeperResourcePath).
		Name(sleeperActual.Name).
		Body(&meta_v1.DeleteOptions{
			Preconditions: &meta_v1.Preconditions{
				UID: &sleeperActual.UID,
			},
		}).
		Do().
		Error()
	require.NoError(t, err)

	// Bundle should reach Ready=true state
	AssertBundle(t, ctxTest, cfg.Store, cfg.Namespace, cfg.Bundle)

	// ConfigMap should exist by now
	cmActual, err = cmClient.Get(cm.Name, meta_v1.GetOptions{})
	require.NoError(t, err)

	// Sleeper should have BlockOwnerDeletion updated
	err = sClient.Get().
		Namespace(cfg.Namespace).
		Resource(tprattribute.SleeperResourcePath).
		Name(sleeperActual.Name).
		Do().
		Into(sleeperActual)
	require.NoError(t, err)
}
