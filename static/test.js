url = 'ws://139.129.212.226:8000/ws';
c = new WebSocket(url);

c.onmessage = function(msg){
    var obj = JSON.parse(msg.data);
    $('#players').text(obj);
}

c.onopen = function(){
    c.send("666666")
}