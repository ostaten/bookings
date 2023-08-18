function SelectRoomFromOptions(roomID) {
  let tkn = document.querySelector('meta[name="csrf-token"]').content

  document.getElementById("check-availability-button").addEventListener("click", function(){
    let html = `
    <form id="check-availability-form" action="" method="post" novalidate class="needs-validation">
      <div class="row">
        <div class="col">
          <div class="row" id="reservation-dates-modal">
            <div class="col">
              <input required disabled class="form-control" type="text" name="start" id="start" placeholder="Arrival" autocomplete="off">
            </div>
            <div class="col">
              <input required disabled class="form-control" type="text" name="end" id="end" placeholder="Departure">
            </div>
          </div>
        </div>
      </div>
    </form>
    `
    attention.custom({
      msg: html, 
      title: "Choose your dates",
  
      willOpen: () => {
        const elem = document.getElementById("reservation-dates-modal");
        const rp = new DateRangePicker(elem, {
          format: 'yyyy-mm-dd',
          showOnFocus: true,
          minDate: new Date(),
        })							
      },
  
      didOpen: () => {
        document.getElementById("start").removeAttribute('disabled');
        document.getElementById("end").removeAttribute('disabled');
      },
      
      callback: async function(result) {
        console.log('called');
  
        let form = document.getElementById("check-availability-form");
        let formData = new FormData(form);
        formData.append("csrf_token", tkn);
        formData.append("room_id", roomID)
        console.log(formData.get("room_id"))
        const response = await fetch('/search-availability-json', {
          method: "post",
          body: formData,
        });
        const data = await response.json();
        console.log(response);
        console.log(data.ok);
        console.log(data.message);
        if (data.ok) {
          attention.custom({
            icon: 'success',
            msg: '<p>Room is available!</p>'
               + '<p><a href="/book-room?id='
               + data.room_id 
               + '&s=' 
               + data.start_date
               + '&e='
               + data.end_date
               + '" class="btn btn-primary">'
               + 'Book now!</a></p>',
            showConfirmButton: false,
          });
        } else {
          attention.error({
            msg: "No availability",
          });
        }
      }
    });
  });  
}

