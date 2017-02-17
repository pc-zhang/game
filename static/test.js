
url = 'ws://139.129.212.226:8000/ws';
c = new WebSocket(url);
$("#btn").attr({"disabled":"disabled"});
$("#btn").click(function(){
    var mydate = new Date();
    start = mydate.getTime();
});
$("#btn").click(function(){
    var mydate = new Date();
    start = mydate.getTime();
});

function randomString(len) {
　　len = len || 32;
　　var $chars = 'ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678';    /****默认去掉了容易混淆的字符oOLl,9gq,Vv,Uu,I1****/
　　var maxPos = $chars.length;
　　var pwd = '';
　　for (i = 0; i < len; i++) {
　　　　pwd += $chars.charAt(Math.floor(Math.random() * maxPos));
　　}
　　return pwd;
}

var obj;
c.onmessage = function(msg){
    obj = JSON.parse(msg.data);
    if(obj.roomState.currentPlayer==reg["playerName"]){
        $("#btn").removeAttr("disabled");
    }else{
        $("#btn").attr({"disabled":"disabled"});
    }
    if(obj.type=="tick"){
        $("#ticks").text(obj.tick);
    }else{
        $("#ticks").text("");
    }
    players = obj.roomState.playerList;
    $("#players").empty();
    for(var i=0;i<players.length;i++){
        if(obj.roomState.currentPlayer==players[i]){
            $("#players").append("<li>->"+players[i]+"</li>");
        }else{
            $("#players").append("<li>"+players[i]+"</li>");
        }
    }
    auditors = obj.roomState.auditorList;
    $("#auditors").empty();
    for(var i=0;i<auditors.length;i++){
        $("#auditors").append("<li>"+auditors[i]+"</li>");
    }

    console.log(obj);
}
reg = {}
c.onopen = function(){
    reg["roomNum"]="666666";
    reg["playerName"]=randomString(8);
    $("#you").text(reg["playerName"]);
    c.send(JSON.stringify(reg));
}
