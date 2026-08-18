function disableall()
{
   i = 0;
   while(i < btnarray.length)
   {
      btnarray[i].lockup();
      i++;
   }
   btn_back.useHandCursor = false;
   islocked = true;
   lockup._alpha = 100;
}
function enableall()
{
   i = 0;
   while(i < btnarray.length)
   {
      btnarray[i].lockup2();
      i++;
   }
   btn_back.useHandCursor = true;
   islocked = false;
   lockup._alpha = 0;
}
function checkifvalid(input)
{
   returnvalue = false;
   switch(input)
   {
      case 8:
         returnvalue = true;
         break;
      case 13:
         returnvalue = true;
         break;
      case 16:
         returnvalue = true;
         break;
      case 17:
         returnvalue = true;
         break;
      case 20:
         returnvalue = true;
         break;
      case 33:
         returnvalue = true;
         break;
      case 34:
         returnvalue = true;
         break;
      case 35:
         returnvalue = true;
         break;
      case 36:
         returnvalue = true;
         break;
      case 37:
         returnvalue = true;
         break;
      case 38:
         returnvalue = true;
         break;
      case 39:
         returnvalue = true;
         break;
      case 40:
         returnvalue = true;
         break;
      case 45:
         returnvalue = true;
         break;
      case 46:
         returnvalue = true;
         break;
      case 145:
         returnvalue = true;
         break;
      case 96:
         returnvalue = true;
         break;
      case 97:
         returnvalue = true;
         break;
      case 98:
         returnvalue = true;
         break;
      case 99:
         returnvalue = true;
         break;
      case 100:
         returnvalue = true;
         break;
      case 101:
         returnvalue = true;
         break;
      case 102:
         returnvalue = true;
         break;
      case 103:
         returnvalue = true;
         break;
      case 104:
         returnvalue = true;
         break;
      case 105:
         returnvalue = true;
         break;
      case 106:
         returnvalue = true;
         break;
      case 107:
         returnvalue = true;
         break;
      case 109:
         returnvalue = true;
         break;
      case 110:
         returnvalue = true;
         break;
      case 111:
         returnvalue = true;
         break;
      case 113:
         returnvalue = true;
         break;
      case 115:
         returnvalue = true;
         break;
      case 118:
         returnvalue = true;
         break;
      case 119:
         returnvalue = true;
         break;
      case 120:
         returnvalue = true;
         break;
      case 121:
         returnvalue = true;
         break;
      case 123:
         returnvalue = true;
         break;
      case 186:
         returnvalue = true;
         break;
      case 187:
         returnvalue = true;
         break;
      case 188:
         returnvalue = true;
         break;
      case 189:
         returnvalue = true;
         break;
      case 190:
         returnvalue = true;
         break;
      case 191:
         returnvalue = true;
         break;
      case 192:
         returnvalue = true;
         break;
      case 219:
         returnvalue = true;
         break;
      case 220:
         returnvalue = true;
         break;
      case 221:
         returnvalue = true;
         break;
      case 222:
         returnvalue = true;
   }
   if(input >= 65 && input <= 90 || input >= 48 && input <= 57)
   {
      returnvalue = true;
   }
   return returnvalue;
}
btn_back.onRelease = function()
{
   if(!_root.fadeaway && !islocked)
   {
      btn_back.useHandCursor = false;
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 10;
      _root.gotomenu = true;
      _root.playsound("menu.wav");
   }
};
lockup._alpha = 0;
btnarray = new Array();
islocked = false;
getinput = -1;
keyListener = new Object();
keyListener.onKeyDown = function()
{
   if(islocked && checkifvalid(Key.getCode()))
   {
      islocked = false;
      getinput = Key.getCode();
      enableall();
      targetkey.taketheinput();
   }
};
Key.addListener(keyListener);
if(_root.savedata2.data.musicON)
{
   music1.gotoAndStop(3);
}
if(!_root.savedata2.data.musicON)
{
   music2.gotoAndStop(3);
}
music1.onRelease = function()
{
   if(music1._currentframe != 3)
   {
      _root.savedata2.data.musicON = true;
      music1.gotoAndStop(3);
      music2.gotoAndStop(1);
   }
};
music2.onRelease = function()
{
   if(music2._currentframe != 3)
   {
      _root.savedata2.data.musicON = false;
      music2.gotoAndStop(3);
      music1.gotoAndStop(1);
      _root.stopallmusic();
   }
};
if(_root.savedata2.data.soundON)
{
   sound1.gotoAndStop(3);
}
if(!_root.savedata2.data.soundON)
{
   sound2.gotoAndStop(3);
}
sound1.onRelease = function()
{
   if(sound1._currentframe != 3)
   {
      _root.savedata2.data.soundON = true;
      sound1.gotoAndStop(3);
      sound2.gotoAndStop(1);
   }
};
sound2.onRelease = function()
{
   if(sound2._currentframe != 3)
   {
      _root.savedata2.data.soundON = false;
      sound2.gotoAndStop(3);
      sound1.gotoAndStop(1);
   }
};
if(_root.savedata2.data.def_quality == 1)
{
   quality1.gotoAndStop(3);
}
if(_root.savedata2.data.def_quality == 2)
{
   quality2.gotoAndStop(3);
}
if(_root.savedata2.data.def_quality == 3)
{
   quality3.gotoAndStop(3);
}
quality1.onRelease = function()
{
   if(quality1._currentframe != 3)
   {
      _root.savedata2.data.def_quality = 1;
      quality1.gotoAndStop(3);
      quality2.gotoAndStop(1);
      quality3.gotoAndStop(1);
   }
};
quality2.onRelease = function()
{
   if(quality2._currentframe != 3)
   {
      _root.savedata2.data.def_quality = 2;
      quality2.gotoAndStop(3);
      quality1.gotoAndStop(1);
      quality3.gotoAndStop(1);
   }
};
quality3.onRelease = function()
{
   if(quality3._currentframe != 3)
   {
      _root.savedata2.data.def_quality = 3;
      quality3.gotoAndStop(3);
      quality1.gotoAndStop(1);
      quality2.gotoAndStop(1);
   }
};
