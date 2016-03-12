$(document).ready(function() {
  var socket = io();

  socket.on('new_room', function(msg) {
    var b = $('div.chat-window');
    b.addClass("chat-window")
      .removeClass('default')
      .removeClass('closed')
      .addClass(msg.Room)
      .data('room', msg.Room);
    b.find('input.message').focus();

    window.localStorage.room = msg.Room;

    var messages = $("div.chat-window."+msg.Room).find('.messages');

    // Retrieve old messages
    $.getJSON('/api/rooms/' + msg.Room, function(data) {
      $.each(data.Messages, function(i, v) {
        var msg = $("<div class='msg'><div class='name'></div><div class='text'></div></div>");
        $(msg).find('.name').html(v.Author);
        $(msg).find('.text').html(v.Text);
        messages.append(msg);
      });
      messages.scrollTop(10000);
    });
  });

  socket.on('message', function(m2) {
    var messages = $("div.chat-window."+m2.Room).find('.messages');
    if (m2.Room) {
      var msg = $("<div class='msg'><div class='name'></div><div class='text'></div></div>");
      $(msg).find('.name').html(m2.Name);
      $(msg).find('.text').html(m2.Msg);
      messages.append(msg);
      messages.scrollTop(10000);
    }
  });

  $(document).on('submit', '.chat-window form', function() {
    var data = {
      Room: $(this).parent().data('room'),
      Msg:  $(this).find('.message').val()
    };
    socket.emit('message', data);
    $(this).find('.message').val('');
    return false;
  });

  $(document).on('click', '.chat-window.closed', function() {
    socket.emit("chat_started", { Room: window.localStorage.room });
    $(this).removeClass('closed').addClass('open');
    window.localStorage.open = true;
  });

  $(document).on('click', '.chat-window.open h3', function() {
    window.localStorage.open = false;
    $(this).parent().addClass('closed').removeClass('open');
  });

  if (window.localStorage.open === 'true') {
    socket.emit("chat_started", { Room: window.localStorage.room });
    $('.chat-window').removeClass('closed').addClass('open');
  } else {
    $('.chat-window').addClass('closed').removeClass('open');
  }

});

