package utils

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// GetDockerVersion предоставляет ворсию docker сервиса, если он запущен на компе.
func GetDockerVersion() (string, error) {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	cli.NegotiateAPIVersion(ctx)
	defer cli.Close()

	if err != nil {
		return "-", errors.New("Ошибка создания клиента сервиса docker. " + err.Error())
	}

	_, errInfo := cli.Info(ctx)
	if errInfo != nil {
		return "-", errors.New("Ошибка создания клиента сервиса docker. " + errInfo.Error())
	}

	return cli.ClientVersion(), nil
}

// GetDockerImageVersions предоставляет список версий docker образов по названию.
// Название должно включать в себя имя репозитория (например di.topaz-atcs.com/fmtp_channel)
func GetDockerImageVersions(imageName string) ([]string, error) {
	var retValue []string
	var retErr error = nil

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	cli.NegotiateAPIVersion(ctx)
	defer cli.Close()

	images, err := cli.ImageList(ctx, types.ImageListOptions{All: false})
	if err != nil {
		retErr = errors.New("Ошибка запроса списка образов docker. " + err.Error())
		return retValue, retErr
	}

	for _, img := range images {
	TAGLBL:
		for _, tag := range img.RepoTags {
			if strings.Contains(tag, imageName) == true {
				if ind := strings.Index(tag, ":"); ind != -1 {
					retValue = append(retValue, tag[ind+1:])
					break TAGLBL
				}
			}
		}
	}

	return retValue, retErr
}

// DockerPullImages выкачивает docker образы по названию.
// Название должно включать в себя имя репозитория (например di.topaz-atcs.com/fmtp_channel)
func DockerPullImages(imageName string) error {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	cli.NegotiateAPIVersion(ctx)
	defer cli.Close()

	if _, err = cli.ImagePull(ctx, imageName, types.ImagePullOptions{All: true}); err != nil {
		return err
	}

	return nil
}

// StopAndRmContainers остановка и удаление контейнеров по названию образа
func StopAndRmContainers(imageName string) error {
	var retErr error = nil
	var stopDur time.Duration = 100 * time.Millisecond

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	cli.NegotiateAPIVersion(ctx)
	defer cli.Close()
	if err != nil {
		return errors.New("Ошибка создания клиента сервиса docker. " + err.Error())
	}

	//args := filters.NewArgs(filters.Arg("reference", containerNameMask))

	cnts, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return errors.New("Ошибка запроса списка контейнеров docker. " + err.Error())
	}

	for _, cnt := range cnts {
		if strings.Contains(cnt.Image, imageName) {
			if stopErr := cli.ContainerStop(ctx, cnt.ID, &stopDur); stopErr != nil {
				return fmt.Errorf("Ошибка остановки docker контейнера. Ошибка: %s", err.Error())
			} else {
				if rmErr := cli.ContainerRemove(ctx, cnt.ID, types.ContainerRemoveOptions{}); rmErr != nil {
					return fmt.Errorf("Ошибка удаления docker контейнера. Ошибка: %s", rmErr.Error())
				}
			}
		}
	}
	return retErr
}
