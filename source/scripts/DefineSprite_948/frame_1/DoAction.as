function startmission()
{
   if(menu1._currentframe == menu1._totalframes)
   {
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
      totalplayers = 0;
      if(_root.p1ptype > 0)
      {
         totalplayers += 1;
      }
      if(_root.p2ptype > 0)
      {
         totalplayers += 1;
      }
      if(totalplayers >= 1)
      {
         _root.totallives = 10;
         _root.mapnumber = 5;
         _root.campaignmode = true;
         newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
         newmc.targetframe = 10;
         _root.gototest = false;
      }
      else
      {
         warning.play();
      }
   }
}
function WRITE_DATA()
{
   _root.savedata3.data.p1name = menu1.inputname.text;
   _root.savedata3.data.p1color = menu1.colornumber;
   _root.savedata3.data.p1shirt = menu1.player.shirt._currentframe;
   _root.savedata3.data.p1hat = menu1.player.hat._currentframe;
   _root.savedata3.data.p1gun = menu1.player.gundisplay._currentframe;
   _root.savedata3.data.p1perk = menu1.perkdisplay._currentframe;
   _root.savedata3.data.p1ptype = menu1.playertype;
   _root.savedata3.data.p1team = menu1.teamselect.team;
   _root.savedata3.data.p2name = menu2.inputname.text;
   _root.savedata3.data.p2color = menu2.colornumber;
   _root.savedata3.data.p2shirt = menu2.player.shirt._currentframe;
   _root.savedata3.data.p2hat = menu2.player.hat._currentframe;
   _root.savedata3.data.p2gun = menu2.player.gundisplay._currentframe;
   _root.savedata3.data.p2perk = menu2.perkdisplay._currentframe;
   _root.savedata3.data.p2ptype = menu2.playertype;
   _root.savedata3.data.p2team = menu2.teamselect.team;
}
function gotolevel(input)
{
   phase = 2;
   infopanel.number = input;
   _root.campaignlevel = input;
}
btn_back.onRelease = function()
{
   if(!_root.fadeaway)
   {
      if(phase == 1)
      {
         btn_back.useHandCursor = false;
         newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
         newmc.targetframe = 10;
         _root.gotomenu = true;
      }
      else if(phase == 3)
      {
         phase = 4;
      }
      _root.playsound("menu.wav");
   }
};
phase = 1;
menu1._alpha = 0;
menu2._alpha = 0;
infopanel._alpha = 0;
slider._x = _root.slideprevx;
vx = 0;
if(_root.savedata3.data.levelarray[1] == 2 && _root.savedata3.data.levelarray[2] == 0)
{
   _root.savedata3.data.levelarray[2] = 1;
}
if(_root.savedata3.data.levelarray[2] == 2 && _root.savedata3.data.levelarray[3] == 0)
{
   _root.savedata3.data.levelarray[3] = 1;
}
if(_root.savedata3.data.levelarray[3] == 2 && _root.savedata3.data.levelarray[4] == 0)
{
   _root.savedata3.data.levelarray[4] = 1;
}
if(_root.savedata3.data.levelarray[4] == 2 && _root.savedata3.data.levelarray[5] == 0)
{
   _root.savedata3.data.levelarray[5] = 1;
}
if(_root.savedata3.data.levelarray[5] == 2 && _root.savedata3.data.levelarray[6] == 0)
{
   _root.savedata3.data.levelarray[6] = 1;
}
if(_root.savedata3.data.levelarray[6] == 2 && _root.savedata3.data.levelarray[7] == 0)
{
   _root.savedata3.data.levelarray[7] = 1;
}
if(_root.savedata3.data.levelarray[7] == 2 && _root.savedata3.data.levelarray[8] == 0)
{
   _root.savedata3.data.levelarray[8] = 1;
}
if(_root.savedata3.data.levelarray[8] == 2 && _root.savedata3.data.levelarray[9] == 0)
{
   _root.savedata3.data.levelarray[9] = 1;
}
if(_root.showunlocks != 0)
{
   show_unlock.gotoAndStop(_root.showunlocks);
   show_unlock._alpha = 100;
}
else
{
   show_unlock._alpha = 0;
   show_unlock.swapDepths(1);
   removeMovieClip(show_unlock);
}
this.onEnterFrame = function()
{
   if(phase == 1)
   {
      if(_ymouse > 100 && _ymouse < 510 && _root.showunlocks == 0)
      {
         if(_xmouse < 150 && slider._x < 0)
         {
            vx += (150 - _xmouse) / 8;
         }
         if(_xmouse > 750 && slider._x > -857)
         {
            vx -= (_xmouse - 750) / 8;
         }
      }
      vx *= 0.8;
      slider._x += vx;
      if(slider._x > 0)
      {
         slider._x = 0;
         vx = 0;
      }
      if(slider._x < -857)
      {
         vx = 0;
         slider._x = -857;
      }
      slider._x = Math.round(slider._x);
      _root.slideprevx = slider._x;
   }
   else if(phase == 2)
   {
      if(slider && slider._alpha > 1)
      {
         slider._alpha -= 20;
      }
      if(slider && slider._alpha <= 1)
      {
         slider.swapDepths(1);
         removeMovieClip(slider);
         delete slider.onEnterFrame;
      }
      if(menu1._currentframe == 1)
      {
         menu1.gotoAndStop(menu1._totalframes);
         menu2.gotoAndStop(menu2._totalframes);
         infopanel.gotoAndStop(2);
      }
      if(menu1._alpha < 100)
      {
         menu1._alpha += 20;
         menu2._alpha = menu1._alpha;
         infopanel._alpha = menu1._alpha;
         if(menu1._alpha >= 100)
         {
            phase = 3;
         }
      }
   }
   else if(phase == 4)
   {
      if(!slider)
      {
         asdf = this.attachMovie("menuc_slider","slider",2);
         asdf._alpha = 0;
         asdf._x = _root.slideprevx;
      }
      if(menu1._alpha > 1)
      {
         menu1._alpha -= 20;
         menu2._alpha = menu1._alpha;
         infopanel._alpha = menu1._alpha;
         asdf._alpha += 20;
      }
      else
      {
         menu1._alpha = 0;
         menu2._alpha = 0;
         infopanel._alpha = 0;
         asdf._alpha = 100;
         menu1.gotoAndStop(1);
         menu2.gotoAndStop(1);
         infopanel.gotoAndStop(1);
         phase = 1;
      }
   }
};
