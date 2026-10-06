package main

func redirectURL(shortURL string) (string, error) {

	url, err := findURL(shortURL)
	if err != nil {
		return "", err
	}
	return url.LongURL, nil
}
