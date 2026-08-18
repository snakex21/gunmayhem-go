function modeallplay()
{
   if(modebtn1._currentframe != 1)
   {
      modebtn1.play();
   }
   if(modebtn2._currentframe != 1)
   {
      modebtn2.play();
   }
   if(modebtn3._currentframe != 1)
   {
      modebtn3.play();
   }
   if(modebtn4._currentframe != 1)
   {
      modebtn4.play();
   }
   if(modebtn5._currentframe != 1)
   {
      modebtn5.play();
   }
   if(modebtn6._currentframe != 1)
   {
      modebtn6.play();
   }
}
function modealldisable()
{
   modebtn1.chosen._alpha = 0;
   modebtn2.chosen._alpha = 0;
   modebtn3.chosen._alpha = 0;
   modebtn4.chosen._alpha = 0;
   modebtn5.chosen._alpha = 0;
   modebtn6.chosen._alpha = 0;
}
function allplay()
{
   if(mapbtn0._currentframe != 1)
   {
      mapbtn0.play();
   }
   if(mapbtn1._currentframe != 1)
   {
      mapbtn1.play();
   }
   if(mapbtn2._currentframe != 1)
   {
      mapbtn2.play();
   }
   if(mapbtn3._currentframe != 1)
   {
      mapbtn3.play();
   }
   if(mapbtn4._currentframe != 1)
   {
      mapbtn4.play();
   }
   if(mapbtn5._currentframe != 1)
   {
      mapbtn5.play();
   }
   if(mapbtn6._currentframe != 1)
   {
      mapbtn6.play();
   }
   if(mapbtn7._currentframe != 1)
   {
      mapbtn7.play();
   }
   if(mapbtn8._currentframe != 1)
   {
      mapbtn8.play();
   }
   if(mapbtn9._currentframe != 1)
   {
      mapbtn9.play();
   }
   if(mapbtn10._currentframe != 1)
   {
      mapbtn10.play();
   }
   if(mapbtn11._currentframe != 1)
   {
      mapbtn11.play();
   }
   if(mapbtn12._currentframe != 1)
   {
      mapbtn12.play();
   }
}
function alldisable()
{
   mapbtn0.chosen._alpha = 0;
   mapbtn1.chosen._alpha = 0;
   mapbtn2.chosen._alpha = 0;
   mapbtn3.chosen._alpha = 0;
   mapbtn4.chosen._alpha = 0;
   mapbtn5.chosen._alpha = 0;
   mapbtn6.chosen._alpha = 0;
   mapbtn7.chosen._alpha = 0;
   mapbtn8.chosen._alpha = 0;
   mapbtn9.chosen._alpha = 0;
   mapbtn10.chosen._alpha = 0;
   mapbtn11.chosen._alpha = 0;
   mapbtn12.chosen._alpha = 0;
}
function WRITE_DATA()
{
   savedata.data.mapnumber = mapdisplay._currentframe;
   savedata.data.gamemode = modedisplay._currentframe;
   savedata.data.p1name = menu1.inputname.text;
   savedata.data.p1color = menu1.colornumber;
   savedata.data.p1shirt = menu1.player.shirt._currentframe;
   savedata.data.p1hat = menu1.player.hat._currentframe;
   savedata.data.p1gun = menu1.player.gundisplay._currentframe;
   savedata.data.p1perk = menu1.perkdisplay._currentframe;
   savedata.data.p1ptype = menu1.playertype;
   savedata.data.p1team = menu1.teamselect.team;
   savedata.data.p2name = menu2.inputname.text;
   savedata.data.p2color = menu2.colornumber;
   savedata.data.p2shirt = menu2.player.shirt._currentframe;
   savedata.data.p2hat = menu2.player.hat._currentframe;
   savedata.data.p2gun = menu2.player.gundisplay._currentframe;
   savedata.data.p2perk = menu2.perkdisplay._currentframe;
   savedata.data.p2ptype = menu2.playertype;
   savedata.data.p2team = menu2.teamselect.team;
   savedata.data.p3name = menu3.inputname.text;
   savedata.data.p3color = menu3.colornumber;
   savedata.data.p3shirt = menu3.player.shirt._currentframe;
   savedata.data.p3hat = menu3.player.hat._currentframe;
   savedata.data.p3gun = menu3.player.gundisplay._currentframe;
   savedata.data.p3perk = menu3.perkdisplay._currentframe;
   savedata.data.p3ptype = menu3.playertype;
   savedata.data.p3team = menu3.teamselect.team;
   savedata.data.p4name = menu4.inputname.text;
   savedata.data.p4color = menu4.colornumber;
   savedata.data.p4shirt = menu4.player.shirt._currentframe;
   savedata.data.p4hat = menu4.player.hat._currentframe;
   savedata.data.p4gun = menu4.player.gundisplay._currentframe;
   savedata.data.p4perk = menu4.perkdisplay._currentframe;
   savedata.data.p4ptype = menu4.playertype;
   savedata.data.p4team = menu4.teamselect.team;
}
savedata = SharedObject.getLocal("arenagamedata");
if(!savedata.data.filled)
{
   savedata.data.filled = true;
   savedata.data.mapnumber = mapdisplay._totalframes;
   savedata.data.gamemode = modedisplay._totalframes;
   savedata.data.p1name = "Player 1";
   savedata.data.p1color = 2;
   savedata.data.p1shirt = 1;
   savedata.data.p1hat = 1;
   savedata.data.p1gun = 1;
   savedata.data.p1perk = 7;
   savedata.data.p1ptype = 1;
   savedata.data.p2name = "Player 2";
   savedata.data.p2color = 5;
   savedata.data.p2shirt = 1;
   savedata.data.p2hat = 1;
   savedata.data.p2gun = 1;
   savedata.data.p2perk = 7;
   savedata.data.p2ptype = 1;
   savedata.data.p3name = "Player 3";
   savedata.data.p3color = 8;
   savedata.data.p3shirt = 1;
   savedata.data.p3hat = 1;
   savedata.data.p3gun = 1;
   savedata.data.p3perk = 7;
   savedata.data.p3ptype = 0;
   savedata.data.p4name = "Player 4";
   savedata.data.p4color = 10;
   savedata.data.p4shirt = 1;
   savedata.data.p4hat = 1;
   savedata.data.p4gun = 1;
   savedata.data.p4perk = 7;
   savedata.data.p4ptype = 0;
   savedata.data.p1team = 1;
   savedata.data.p2team = 2;
   savedata.data.p3team = 1;
   savedata.data.p4team = 2;
}
_X = 1800;
frame = 3;
_root.mapnumber = 0;
mapdisplay.gotoAndStop(_root.mapnumber);
if(_root.mapnumber == 0)
{
   mapdisplay.gotoAndStop(mapdisplay._totalframes);
}
_root.gamemode = 1;
modedisplay.gotoAndStop(modedisplay._totalframes);
alldisable();
modealldisable();
mapdisplay.gotoAndStop(savedata.data.mapnumber);
_root.mapnumber = mapdisplay._currentframe;
if(mapdisplay._currentframe == mapdisplay._totalframes)
{
   _root.mapnumber = 0;
}
modedisplay.gotoAndStop(savedata.data.gamemode);
_root.gamemode = modedisplay._totalframes - savedata.data.gamemode + 1;
switch(mapdisplay._totalframes - savedata.data.mapnumber + 1)
{
   case 1:
      mapbtn0.chosen._alpha = 100;
      break;
   case 2:
      mapbtn1.chosen._alpha = 100;
      break;
   case 3:
      mapbtn2.chosen._alpha = 100;
      break;
   case 4:
      mapbtn3.chosen._alpha = 100;
      break;
   case 5:
      mapbtn4.chosen._alpha = 100;
      break;
   case 6:
      mapbtn5.chosen._alpha = 100;
      break;
   case 7:
      mapbtn6.chosen._alpha = 100;
      break;
   case 8:
      mapbtn7.chosen._alpha = 100;
      break;
   case 9:
      mapbtn8.chosen._alpha = 100;
      break;
   case 10:
      mapbtn9.chosen._alpha = 100;
      break;
   case 11:
      mapbtn10.chosen._alpha = 100;
      break;
   case 12:
      mapbtn11.chosen._alpha = 100;
      break;
   case 13:
      mapbtn12.chosen._alpha = 100;
}
switch(modedisplay._currentframe)
{
   case 1:
      modebtn6.chosen._alpha = 100;
      break;
   case 2:
      modebtn5.chosen._alpha = 100;
      break;
   case 3:
      modebtn4.chosen._alpha = 100;
      break;
   case 4:
      modebtn3.chosen._alpha = 100;
      break;
   case 5:
      modebtn2.chosen._alpha = 100;
      break;
   case 6:
      modebtn1.chosen._alpha = 100;
}
this.onEnterFrame = function()
{
   if(frame == 1)
   {
      _X = _X + (- _X) / 3;
   }
   else if(frame == 2)
   {
      _X = _X + (900 - _X) / 3;
   }
   else if(frame == 3)
   {
      _X = _X + (1800 - _X) / 3;
   }
   if(_X % 1 != 0)
   {
      _X = Math.round(_X);
   }
};
btn_continue.onRelease = function()
{
   _root.playsound("menu.wav");
   frame = 1;
   menu1.play();
   menu2.play();
   menu3.play();
   menu4.play();
};
btn_continue2.onRelease = function()
{
   _root.playsound("menu.wav");
   if(modedisplay._currentframe < 4)
   {
      frame = 2;
   }
   else if(parseInt(modedisplay.inputlives.text) > 0)
   {
      frame = 2;
   }
   else
   {
      warning1.play();
   }
};
btn_back.onRelease = function()
{
   _root.playsound("menu.wav");
   WRITE_DATA();
   frame = 2;
   menu1.gotoAndStop(1);
   menu2.gotoAndStop(1);
   menu3.gotoAndStop(1);
   menu4.gotoAndStop(1);
};
btn_back2.onRelease = function()
{
   _root.playsound("menu.wav");
   frame = 3;
};
btn_back3.onRelease = function()
{
   if(!_root.fadeaway)
   {
      _root.playsound("menu.wav");
      btn_back3.useHandCursor = false;
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 10;
      _root.gotomenu = true;
   }
};
btn_start.onRelease = function()
{
   if(!_root.fadeaway && menu1._currentframe == menu1._totalframes)
   {
      if(_root.mapnumber == 0)
      {
         _root.mapnumber = random(mapdisplay._totalframes - 1) + 1;
      }
      WRITE_DATA();
      _root.p1name = menu1.inputname.text;
      _root.p1color = menu1.colornumber;
      _root.p1shirt = menu1.player.shirt._currentframe;
      _root.p1hat = menu1.player.hat._currentframe;
      _root.p1gun = menu1.player.gundisplay._currentframe;
      _root.p1perk = menu1.perkdisplay._currentframe;
      _root.p1ptype = menu1.playertype;
      _root.p1team = menu1.teamselect.team;
      _root.p2name = menu2.inputname.text;
      _root.p2color = menu2.colornumber;
      _root.p2shirt = menu2.player.shirt._currentframe;
      _root.p2hat = menu2.player.hat._currentframe;
      _root.p2gun = menu2.player.gundisplay._currentframe;
      _root.p2perk = menu2.perkdisplay._currentframe;
      _root.p2ptype = menu2.playertype;
      _root.p2team = menu2.teamselect.team;
      _root.p3name = menu3.inputname.text;
      _root.p3color = menu3.colornumber;
      _root.p3shirt = menu3.player.shirt._currentframe;
      _root.p3hat = menu3.player.hat._currentframe;
      _root.p3gun = menu3.player.gundisplay._currentframe;
      _root.p3perk = menu3.perkdisplay._currentframe;
      _root.p3ptype = menu3.playertype;
      _root.p3team = menu3.teamselect.team;
      _root.p4name = menu4.inputname.text;
      _root.p4color = menu4.colornumber;
      _root.p4shirt = menu4.player.shirt._currentframe;
      _root.p4hat = menu4.player.hat._currentframe;
      _root.p4gun = menu4.player.gundisplay._currentframe;
      _root.p4perk = menu4.perkdisplay._currentframe;
      _root.p4ptype = menu4.playertype;
      _root.p4team = menu4.teamselect.team;
      _root.campaignmode = false;
      totalplayers = 0;
      if(_root.p1ptype > 0)
      {
         totalplayers += 1;
      }
      if(_root.p2ptype > 0)
      {
         totalplayers += 1;
      }
      if(_root.p3ptype > 0)
      {
         totalplayers += 1;
      }
      if(_root.p4ptype > 0)
      {
         totalplayers += 1;
      }
      if(totalplayers >= 2)
      {
         _root.totallives = modedisplay.inputlives.text;
         btn_start.useHandCursor = false;
         _root.campaignmode = false;
         newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
         newmc.targetframe = 10;
         _root.gototest = false;
      }
      else
      {
         warning2.play();
      }
   }
};
