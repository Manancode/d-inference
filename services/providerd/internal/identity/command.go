package identity

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os/exec"
)

type commandRunner interface {
	Output(name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

type CommandSigner struct {
	command string
	tag     string
	runner  commandRunner
}

type keyToolResponse struct {
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
	Note      string `json:"note"`
}

func NewCommandSigner(command string, tag string) *CommandSigner {
	return &CommandSigner{
		command: command,
		tag:     tag,
		runner:  execRunner{},
	}
}

func (c *CommandSigner) PublicKey() (string, error) {
	response, err := c.run("ensure-signing-key", "--tag", c.tag)
	if err != nil {
		return "", err
	}
	return response.PublicKey, nil
}

func (c *CommandSigner) Sign(payload []byte) (string, error) {
	response, err := c.run("sign", "--tag", c.tag, "--payload-base64", base64.StdEncoding.EncodeToString(payload))
	if err != nil {
		return "", err
	}
	if response.Signature == "" {
		return "", errors.New("key tool returned empty signature")
	}
	return response.Signature, nil
}

func (c *CommandSigner) run(args ...string) (keyToolResponse, error) {
	if c.command == "" {
		return keyToolResponse{}, errors.New("command signer requires a command path")
	}
	raw, err := c.runner.Output(c.command, args...)
	if err != nil {
		return keyToolResponse{}, err
	}
	var response keyToolResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return keyToolResponse{}, err
	}
	if response.PublicKey == "" {
		return keyToolResponse{}, errors.New("key tool returned empty public key")
	}
	return response, nil
}
