function killupdate(mcname, mckiller, mod)
{
   if(_root.gamemode != 5 && _root.campaignlevel != 1)
   {
      newmc = this.attachMovie("hud_killfeed","mc" + this.getNextHighestDepth(),this.getNextHighestDepth());
      newmc.mckiller = mckiller;
      newmc.mcname = mcname;
      if(mckiller != "none")
      {
         if(mod == 2)
         {
            newmc.gotoAndStop(3);
         }
         if(mod == 3)
         {
            newmc.gotoAndStop(4);
         }
         if(mod == 4)
         {
            newmc.gotoAndStop(5);
         }
      }
      else
      {
         newmc.gotoAndStop(2);
      }
      newmc._x = 0;
      feednumber += 1;
      newmc._y = (feednumber - 1) * 30;
      newmc.feednumber = feednumber;
      if(countdown._currentframe != 1)
      {
         countdown.gotoAndStop(1);
         _root._x = 0;
         _root._y = 0;
      }
   }
}
function update()
{
   if(_root.gamemode != 4 && _root.gamemode != 5)
   {
      card1.livedisplay.text = _root.player1.lives;
      card2.livedisplay.text = _root.player2.lives;
      card3.livedisplay.text = _root.player3.lives;
      card4.livedisplay.text = _root.player4.lives;
   }
   else if(_root.gamemode == 4)
   {
      card1.livedisplay.text = _root.player1.currentlevel;
      card2.livedisplay.text = _root.player2.currentlevel;
      card3.livedisplay.text = _root.player3.currentlevel;
      card4.livedisplay.text = _root.player4.currentlevel;
   }
   else if(_root.gamemode == 5)
   {
      card1.livedisplay.text = _root.player1.lives;
      card2.livedisplay.text = _root.player2.lives;
      card3.livedisplay.text = _root.player3.lives;
      card4.livedisplay.text = _root.player4.lives;
      if(_root.player1.lives == 0)
      {
         card1.player.gotoAndStop(2);
      }
      if(_root.player2.lives == 0)
      {
         card2.player.gotoAndStop(2);
      }
      if(_root.player3.lives == 0)
      {
         card3.player.gotoAndStop(2);
      }
      if(_root.player4.lives == 0)
      {
         card4.player.gotoAndStop(2);
      }
   }
   if(card1.livedisplay.text != "0")
   {
      card1.ammodisplay.text = _root.player1.bullets;
   }
   if(card2.livedisplay.text != "0")
   {
      card2.ammodisplay.text = _root.player2.bullets;
   }
   if(card3.livedisplay.text != "0")
   {
      card3.ammodisplay.text = _root.player3.bullets;
   }
   if(card4.livedisplay.text != "0")
   {
      card4.ammodisplay.text = _root.player4.bullets;
   }
   if(card1.livedisplay.text != "0")
   {
      if(_root.player1.bullets > 1000)
      {
         card1.ammodisplay.text = "∞";
      }
   }
   if(card2.livedisplay.text != "0")
   {
      if(_root.player2.bullets > 1000)
      {
         card2.ammodisplay.text = "∞";
      }
   }
   if(card3.livedisplay.text != "0")
   {
      if(_root.player3.bullets > 1000)
      {
         card3.ammodisplay.text = "∞";
      }
   }
   if(card4.livedisplay.text != "0")
   {
      if(_root.player4.bullets > 1000)
      {
         card4.ammodisplay.text = "∞";
      }
   }
   if(card1.livedisplay.text != "0")
   {
      if(card1.gunname.text != _root.player1.hand1.gun.Name)
      {
         card1.gunname.text = _root.player1.hand1.gun.Name;
      }
   }
   if(card2.livedisplay.text != "0")
   {
      if(card2.gunname.text != _root.player2.hand1.gun.Name)
      {
         card2.gunname.text = _root.player2.hand1.gun.Name;
      }
   }
   if(card3.livedisplay.text != "0")
   {
      if(card3.gunname.text != _root.player3.hand1.gun.Name)
      {
         card3.gunname.text = _root.player3.hand1.gun.Name;
      }
   }
   if(card4.livedisplay.text != "0")
   {
      if(card4.gunname.text != _root.player4.hand1.gun.Name)
      {
         card4.gunname.text = _root.player4.hand1.gun.Name;
      }
   }
   if(!_root.player1)
   {
      card1._alpha = 0;
   }
   if(!_root.player2)
   {
      card2._alpha = 0;
   }
   if(!_root.player3)
   {
      card3._alpha = 0;
   }
   if(!_root.player4)
   {
      card4._alpha = 0;
   }
}
this.swapDepths(_root.huddepth);
scrollup = false;
time = 0;
qwerqwer._x = 20;
qwerqwer._y = 20;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _X = - _root._x;
      _Y = - _root._y;
      update();
      if(scrollup)
      {
         time += 1;
         if(time >= 2)
         {
            scrollup = false;
            time = 0;
         }
      }
      if(_root.gamemode == 5)
      {
         if(wavedisplay.wavedisplay.text != _root.zombiewave)
         {
            wavedisplay.wavedisplay.text = _root.zombiewave;
         }
         if(wavedisplay.killeddisplay.text != _root.zombiekilled)
         {
            wavedisplay.killeddisplay.text = _root.zombiekilled;
         }
         if(wavedisplay.killedtotaldisplay.text != _root.zombiekilledtotal)
         {
            wavedisplay.killedtotaldisplay.text = _root.zombiekilledtotal;
         }
      }
   }
};
feednumber = 0;
if(_root.gamemode == 5)
{
   this.attachMovie("hud_wavedisplay","wavedisplay",2);
}
