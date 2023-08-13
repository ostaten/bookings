package forms

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestForm_Valid(t *testing.T) {
	r := httptest.NewRequest("POST", "/whatever", nil)
	form := New(r.PostForm)

	isValid := form.Valid()
	if !isValid {
		t.Error("got invalid when should have been valid")
	}
}

func TestForm_Required(t *testing.T) {
	r := httptest.NewRequest("POST", "/whatever", nil)
	form := New(r.PostForm)

	form.Required("a", "b", "c")
	if form.Valid() {
		t.Error("form shows valid when required fields missing")
	}

	postedData := url.Values{}



	postedData.Add("a", "a")
	postedData.Add("b", "a")
	postedData.Add("c", "a")

	r, _ = http.NewRequest("POST", "/whatever", nil)
	r.PostForm = postedData
	form = New(r.PostForm)
	form.Required("a", "b", "c")
	if !form.Valid() {
		t.Error("shows does not have required fields when it does")
	}

	postedData.Set("a", " ")
	r, _ = http.NewRequest("POST", "/whatever", nil)
	r.PostForm = postedData
	form = New(r.PostForm)
	form.Required("a", "b", "c")
	if form.Valid() {
		t.Error("should have error reflecting empty thing")
	}
}

func TestForm_Has(t *testing.T) {
	r := httptest.NewRequest("POST", "/whatever", nil)
	form := New(r.PostForm)

	isValid := form.Has("a", r)
	if isValid {
		t.Error("got valid when should have taken empty as false")
	}
	postedData := url.Values{}
	postedData.Add("c", "s")
	fmt.Println(postedData.Get("c"))
	r = httptest.NewRequest("POST", "/whatever", nil)
	r.PostForm = postedData
	form = New(r.PostForm)

	isValid = form.Has("c", r)
	if !isValid {
		t.Error("got invalid when should have seen 's'")
	}
}

func TestForm_MinLength(t *testing.T) {
	postedData := url.Values{}
	postedData.Add("a", "a")
	r := httptest.NewRequest("POST", "/whatever", nil)
	r.PostForm = postedData

	form := New(r.PostForm)

	isValid := form.MinLength("a", 1, r)
	if !isValid {
		t.Error("got invalid when equal length should work")
	}

	isValid = form.MinLength("a", 2, r)
	if isValid {
		t.Error("got valid when should have been exception to length")
	}
}

func TestForm_IsEmail(t *testing.T) {
	postedData := url.Values{}

	postedData.Add("b", "b@gmail.com")
	r := httptest.NewRequest("POST", "/whatever", nil)
	r.PostForm = postedData

	form := New(r.PostForm)

	form.IsEmail("b")
	if !form.Valid() {
		t.Error("invalid when email address is good")
	}

	postedData.Add("a", "a")
	r = httptest.NewRequest("POST", "/whatever", nil)
	r.PostForm = postedData

	form = New(r.PostForm)

	form.IsEmail("a")
	if form.Valid() {
		t.Error("got valid when should have shown as invalid email")
	}

}


